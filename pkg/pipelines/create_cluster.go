/*
 Copyright 2021 The KubeSphere Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package pipelines

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path/filepath"

	kubekeyapiv1alpha2 "github.com/kubesphere/kubekey/apis/kubekey/v1alpha2"
	"github.com/kubesphere/kubekey/pkg/artifact"
	"github.com/kubesphere/kubekey/pkg/bootstrap/confirm"
	"github.com/kubesphere/kubekey/pkg/bootstrap/precheck"
	"github.com/kubesphere/kubekey/pkg/certs"
	"github.com/kubesphere/kubekey/pkg/container"
	"github.com/kubesphere/kubekey/pkg/images"
	"github.com/kubesphere/kubekey/pkg/kubernetes"
	"github.com/kubesphere/kubekey/pkg/plugins"
	"github.com/kubesphere/kubekey/pkg/plugins/dns"

	kubekeycontroller "github.com/kubesphere/kubekey/controllers/kubekey"
	"github.com/kubesphere/kubekey/pkg/addons"
	"github.com/kubesphere/kubekey/pkg/binaries"
	"github.com/kubesphere/kubekey/pkg/bootstrap/os"
	"github.com/kubesphere/kubekey/pkg/common"
	"github.com/kubesphere/kubekey/pkg/core/module"
	"github.com/kubesphere/kubekey/pkg/core/pipeline"
	"github.com/kubesphere/kubekey/pkg/etcd"
	"github.com/kubesphere/kubekey/pkg/filesystem"
	"github.com/kubesphere/kubekey/pkg/hooks"
	"github.com/kubesphere/kubekey/pkg/k3s"
	"github.com/kubesphere/kubekey/pkg/kubesphere"
	"github.com/kubesphere/kubekey/pkg/loadbalancer"
	"github.com/kubesphere/kubekey/pkg/plugins/network"
	"github.com/kubesphere/kubekey/pkg/plugins/storage"
)

func NewCreateClusterPipeline(runtime *common.KubeRuntime) error {
	noArtifact := runtime.Arg.Artifact == "" // 在线安装为true, 离线安装为false
	skipPushImages := runtime.Arg.SKipPushImages || noArtifact || (!noArtifact && runtime.Cluster.Registry.PrivateRegistry == "")
	skipLocalStorage := true
	if runtime.Arg.DeployLocalStorage != nil {
		skipLocalStorage = !*runtime.Arg.DeployLocalStorage
	} else if runtime.Cluster.KubeSphere.Enabled {
		skipLocalStorage = false
	}

	m := []module.Module{
		&precheck.GreetingsModule{},                                                                    // 打印提示信息
		&precheck.NodePreCheckModule{},                                                                 // 提示需要安装sudo, curl, openssl, ebtables, socat, ipset, ipvsadm, conntrack, chrony, docker, containerd, showmount, rbd, glusterfs软件
		&confirm.InstallConfirmModule{Skip: runtime.Arg.SkipConfirmCheck},                              // 检查这些软件是安装了，尤其是sudo, socat, conntrack
		&artifact.UnArchiveModule{Skip: noArtifact},                                                    // 校验MD5值，解压缩 kubekey-artifact.tar.gz离线软件安装包，如果是在线安装就跳过
		&os.RepositoryModule{Skip: noArtifact || !runtime.Arg.InstallPackages},                         // 如果是离线安装，那么把iso文件挂在上，安装必要的软件，然后卸载
		&binaries.NodeBinariesModule{},                                                                 // 下载etcd, kubeadm, kubelet, kubectl, helm ,kubecni, docker ,crictl等二进制文件 fixme 如果是离线安装，则不需要这一步，
		&os.ConfigureOSModule{},                                                                        // 新建一个系统初始化脚本，放在/usr/local/bin/kube-scripts目录中，主要有关闭swap, selinux, 修改hosts文件等等
		&kubernetes.StatusModule{},                                                                     // 查看集群状态
		&container.InstallContainerModule{},                                                            // 安装ContainerManger，譬如docker, containerd
		&images.CopyImagesToRegistryModule{Skip: skipPushImages},                                       // 把离线安装包中的镜像，推送到私有仓库中 (todo 所以需要提前执行kk init registry命令安装私有仓库，当然如果已经存在私有仓库，那么无需执行)
		&images.PullModule{Skip: runtime.Arg.SkipPullImages},                                           // 在每个节点上把需要的镜像从私有仓库中拉取下来，譬如 etcd, pause, kube-apiserver, scheduler,proxy, coredns, cni ,clium, flannel等等 （fixme, 这里可以优化一下，根据不同节点的角色不同，拉取镜像，而不是一股脑拉取所有镜像）
		&etcd.PreCheckModule{Skip: runtime.Cluster.Etcd.Type != kubekeyapiv1alpha2.KubeKey},            // 检查etcd的状态
		&etcd.CertsModule{},                                                                            // 准备etcd集群签发证书
		&etcd.InstallETCDBinaryModule{Skip: runtime.Cluster.Etcd.Type != kubekeyapiv1alpha2.KubeKey},   // 安装etcd集群
		&etcd.ConfigureModule{Skip: runtime.Cluster.Etcd.Type != kubekeyapiv1alpha2.KubeKey},           // 配置etcd
		&etcd.BackupModule{Skip: runtime.Cluster.Etcd.Type != kubekeyapiv1alpha2.KubeKey},              // 备份etcd的数据
		&kubernetes.InstallKubeBinariesModule{},                                                        // 安装kubeadm, kubelet, kubectl, helm, kubecni的组件
		&kubernetes.InitKubernetesModule{},                                                             // 初始化集群，生成 ~/.kube/config文件
		&dns.ClusterDNSModule{},                                                                        // 利用已经创建好的集群，部署coredns服务
		&kubernetes.StatusModule{},                                                                     // 查看集群状态
		&kubernetes.JoinNodesModule{},                                                                  // kubeadm join node to master
		&loadbalancer.HaproxyModule{Skip: !runtime.Cluster.ControlPlaneEndpoint.IsInternalLBEnabled()}, // 安装Haproxy，反向代理 api-server
		&network.DeployNetworkPluginModule{},                                                           // 安装网络插件
		&kubernetes.ConfigureKubernetesModule{},                                                        // 利用kubectl给各个节点打上标签
		&filesystem.ChownModule{},                                                                      // 修改 ~/.kube/config的权限
		&certs.AutoRenewCertsModule{Skip: !runtime.Cluster.Kubernetes.EnableAutoRenewCerts()},          // 生成自动更新证书的脚本，放在/usr/local/bin/kube-scripts/k8s-certs-renew.sh文件当中
		&kubernetes.SaveKubeConfigModule{},                                                             // 把kubeconfig文件以configmap的形式保存在k8s集群中
		&plugins.DeployPluginsModule{},                                                                 // 部署容器运行时，譬如kata container
		&addons.AddonsModule{},                                                                         // 安装 config.yaml 中的spec.addons自定义模块，实际上就是通过部署 kubectl apply -f 命令部署的
		&storage.DeployLocalVolumeModule{Skip: skipLocalStorage},                                       // 部署 OpenEBS
		&kubesphere.DeployModule{Skip: !runtime.Cluster.KubeSphere.Enabled},                            // 通过部署ks-installer 安装 kubesphere,
		&kubesphere.CheckResultModule{Skip: !runtime.Cluster.KubeSphere.Enabled},                       // 检查 kubesphere 的安装结果
	}

	p := pipeline.Pipeline{
		Name:            "CreateClusterPipeline",
		Modules:         m,
		Runtime:         runtime,
		ModulePostHooks: []module.PostHookInterface{&hooks.UpdateCRStatusHook{}},
	}
	if err := p.Start(); err != nil {
		return err
	}

	if runtime.Cluster.KubeSphere.Enabled {

		fmt.Print(`Installation is complete.

Please check the result using the command:

	kubectl logs -n kubesphere-system $(kubectl get pod -n kubesphere-system -l 'app in (ks-install, ks-installer)' -o jsonpath='{.items[0].metadata.name}') -f

`)
	} else {
		fmt.Print(`Installation is complete.

Please check the result using the command:
		
	kubectl get pod -A

`)

	}

	if runtime.Arg.InCluster {
		if err := kubekeycontroller.UpdateStatus(runtime); err != nil {
			return err
		}
		kkConfigPath := filepath.Join(runtime.GetWorkDir(), fmt.Sprintf("config-%s", runtime.ObjName))
		if config, err := ioutil.ReadFile(kkConfigPath); err != nil {
			return err
		} else {
			runtime.Kubeconfig = base64.StdEncoding.EncodeToString(config)
			if err := kubekeycontroller.UpdateKubeSphereCluster(runtime); err != nil {
				return err
			}
			if err := kubekeycontroller.SaveKubeConfig(runtime); err != nil {
				return err
			}
		}
	}

	return nil
}

func NewK3sCreateClusterPipeline(runtime *common.KubeRuntime) error {
	noArtifact := runtime.Arg.Artifact == ""
	skipPushImages := runtime.Arg.SKipPushImages || noArtifact || (!noArtifact && runtime.Cluster.Registry.PrivateRegistry == "")
	skipLocalStorage := true
	if runtime.Arg.DeployLocalStorage != nil {
		skipLocalStorage = !*runtime.Arg.DeployLocalStorage
	} else if runtime.Cluster.KubeSphere.Enabled {
		skipLocalStorage = false
	}

	m := []module.Module{
		&precheck.GreetingsModule{},
		&artifact.UnArchiveModule{Skip: noArtifact},
		&os.RepositoryModule{Skip: noArtifact || !runtime.Arg.InstallPackages},
		&binaries.K3sNodeBinariesModule{},
		&os.ConfigureOSModule{},
		&k3s.StatusModule{},
		&etcd.PreCheckModule{Skip: runtime.Cluster.Etcd.Type != kubekeyapiv1alpha2.KubeKey},
		&etcd.CertsModule{},
		&etcd.InstallETCDBinaryModule{Skip: runtime.Cluster.Etcd.Type != kubekeyapiv1alpha2.KubeKey},
		&etcd.ConfigureModule{Skip: runtime.Cluster.Etcd.Type != kubekeyapiv1alpha2.KubeKey},
		&etcd.BackupModule{Skip: runtime.Cluster.Etcd.Type != kubekeyapiv1alpha2.KubeKey},
		&k3s.InstallKubeBinariesModule{},
		&k3s.InitClusterModule{},
		&k3s.StatusModule{},
		&k3s.JoinNodesModule{},
		&images.CopyImagesToRegistryModule{Skip: skipPushImages},
		&loadbalancer.K3sHaproxyModule{Skip: !runtime.Cluster.ControlPlaneEndpoint.IsInternalLBEnabled()},
		&network.DeployNetworkPluginModule{},
		&kubernetes.ConfigureKubernetesModule{},
		&filesystem.ChownModule{},
		&certs.AutoRenewCertsModule{Skip: !runtime.Cluster.Kubernetes.EnableAutoRenewCerts()},
		&k3s.SaveKubeConfigModule{},
		&addons.AddonsModule{},
		&storage.DeployLocalVolumeModule{Skip: skipLocalStorage},
		&kubesphere.DeployModule{Skip: !runtime.Cluster.KubeSphere.Enabled},
		&kubesphere.CheckResultModule{Skip: !runtime.Cluster.KubeSphere.Enabled},
	}

	p := pipeline.Pipeline{
		Name:            "K3sCreateClusterPipeline",
		Modules:         m,
		Runtime:         runtime,
		ModulePostHooks: []module.PostHookInterface{&hooks.UpdateCRStatusHook{}},
	}
	if err := p.Start(); err != nil {
		return err
	}

	if runtime.Cluster.KubeSphere.Enabled {

		fmt.Print(`Installation is complete.

Please check the result using the command:

	kubectl logs -n kubesphere-system $(kubectl get pod -n kubesphere-system -l 'app in (ks-install, ks-installer)' -o jsonpath='{.items[0].metadata.name}') -f   

`)
	} else {
		fmt.Print(`Installation is complete.

Please check the result using the command:
		
	kubectl get pod -A

`)

	}

	if runtime.Arg.InCluster {
		if err := kubekeycontroller.UpdateStatus(runtime); err != nil {
			return err
		}
		kkConfigPath := filepath.Join(runtime.GetWorkDir(), fmt.Sprintf("config-%s", runtime.ObjName))
		if config, err := ioutil.ReadFile(kkConfigPath); err != nil {
			return err
		} else {
			runtime.Kubeconfig = base64.StdEncoding.EncodeToString(config)
			if err := kubekeycontroller.UpdateKubeSphereCluster(runtime); err != nil {
				return err
			}
			if err := kubekeycontroller.SaveKubeConfig(runtime); err != nil {
				return err
			}
		}
	}

	return nil
}

func CreateCluster(args common.Argument, downloadCmd string) error {
	args.DownloadCommand = func(path, url string) string {
		// this is an extension point for downloading tools, for example users can set the timeout, proxy or retry under
		// some poor network environment. Or users even can choose another cli, it might be wget.
		// perhaps we should have a build-in download function instead of totally rely on the external one
		return fmt.Sprintf(downloadCmd, path, url)
	}

	var loaderType string
	if args.FilePath != "" {
		loaderType = common.File
	} else {
		loaderType = common.AllInOne // 没有指定配置文件的情况下，就是在kk二进制文件所在的机器上部署一个单节点的K8S集群
	}

	runtime, err := common.NewKubeRuntime(loaderType, args)
	if err != nil {
		return err
	}
	if args.InCluster {
		c, err := kubekeycontroller.NewKubekeyClient()
		if err != nil {
			return err
		}
		runtime.ClientSet = c
		if err := kubekeycontroller.ClearConditions(runtime); err != nil {
			return err
		}
	}

	switch runtime.Cluster.Kubernetes.Type {
	case common.K3s:
		if err := NewK3sCreateClusterPipeline(runtime); err != nil {
			return err
		}
	case common.Kubernetes:
		if err := NewCreateClusterPipeline(runtime); err != nil {
			return err
		}
	default:
		if err := NewCreateClusterPipeline(runtime); err != nil {
			return err
		}
	}
	return nil
}
