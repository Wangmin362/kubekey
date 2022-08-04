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
	"fmt"

	kubekeyapiv1alpha2 "github.com/kubesphere/kubekey/apis/kubekey/v1alpha2"
	kubekeycontroller "github.com/kubesphere/kubekey/controllers/kubekey"
	"github.com/kubesphere/kubekey/pkg/artifact"
	"github.com/kubesphere/kubekey/pkg/binaries"
	"github.com/kubesphere/kubekey/pkg/bootstrap/confirm"
	"github.com/kubesphere/kubekey/pkg/bootstrap/os"
	"github.com/kubesphere/kubekey/pkg/bootstrap/precheck"
	"github.com/kubesphere/kubekey/pkg/bootstrap/registry"
	"github.com/kubesphere/kubekey/pkg/certs"
	"github.com/kubesphere/kubekey/pkg/common"
	"github.com/kubesphere/kubekey/pkg/container"
	"github.com/kubesphere/kubekey/pkg/core/module"
	"github.com/kubesphere/kubekey/pkg/core/pipeline"
	"github.com/kubesphere/kubekey/pkg/etcd"
	"github.com/kubesphere/kubekey/pkg/filesystem"
	"github.com/kubesphere/kubekey/pkg/hooks"
	"github.com/kubesphere/kubekey/pkg/images"
	"github.com/kubesphere/kubekey/pkg/k3s"
	"github.com/kubesphere/kubekey/pkg/kubernetes"
	"github.com/kubesphere/kubekey/pkg/loadbalancer"
)

func NewAddNodesPipeline(runtime *common.KubeRuntime) error {
	noArtifact := runtime.Arg.Artifact == ""

	m := []module.Module{
		&precheck.GreetingsModule{},                                                                    // 打印标志语
		&precheck.NodePreCheckModule{},                                                                 // 检查sudo ,curl, openssl, ebtables, socat, ipset, ipvasdm, conntrack,chrony, docker, containerd, showmount, rbd, glusterfs等软件是否安装并给出提示
		&confirm.InstallConfirmModule{Skip: runtime.Arg.SkipConfirmCheck},                              // 检查socat, conntrack软件是否安装
		&artifact.UnArchiveModule{Skip: noArtifact},                                                    // 加压缩离线安装包
		&os.RepositoryModule{Skip: noArtifact || !runtime.Arg.InstallPackages},                         // 如果是在线安装，就需要安装镜像仓库
		&binaries.NodeBinariesModule{},                                                                 // 下载etcd, kubeadm, kubelet, kubecni等等二进制软件
		&os.ConfigureOSModule{},                                                                        // 初始化系统的脚本
		&registry.RegistryCertsModule{Skip: len(runtime.GetHostsByRole(common.Registry)) == 0},         // 同步镜像仓库证书
		&kubernetes.StatusModule{},                                                                     // 获取集群状态
		&container.InstallContainerModule{},                                                            // 安装容器管理器，譬如congtainerd, crio
		&images.PullModule{Skip: runtime.Arg.SkipPullImages},                                           // 拉取诸如kube-proxy, coredns, kube-scheduler等等镜像
		&etcd.PreCheckModule{Skip: runtime.Cluster.Etcd.Type != kubekeyapiv1alpha2.KubeKey},            // 检测etcd的状态
		&etcd.CertsModule{},                                                                            // 同步etcd证书
		&etcd.InstallETCDBinaryModule{Skip: runtime.Cluster.Etcd.Type != kubekeyapiv1alpha2.KubeKey},   // 安装etcd
		&etcd.ConfigureModule{Skip: runtime.Cluster.Etcd.Type != kubekeyapiv1alpha2.KubeKey},           // 生成etcd配置文件
		&etcd.BackupModule{Skip: runtime.Cluster.Etcd.Type != kubekeyapiv1alpha2.KubeKey},              // etcd的备份功能
		&kubernetes.InstallKubeBinariesModule{},                                                        // 安装服务，譬如kubelet
		&kubernetes.JoinNodesModule{},                                                                  // 利用kubeadm加入节点
		&loadbalancer.HaproxyModule{Skip: !runtime.Cluster.ControlPlaneEndpoint.IsInternalLBEnabled()}, // 安装haproxy，反向代理 api-server
		&kubernetes.ConfigureKubernetesModule{},                                                        // 利用 kubectl 给各个节点打上标签
		&filesystem.ChownModule{},                                                                      // 修改 ~/.kube/config的uid gid
		&certs.AutoRenewCertsModule{Skip: !runtime.Cluster.Kubernetes.EnableAutoRenewCerts()},          // 生成自动更新证书的脚本
	}

	p := pipeline.Pipeline{
		Name:            "AddNodesPipeline",
		Modules:         m,
		Runtime:         runtime,
		ModulePostHooks: []module.PostHookInterface{&hooks.UpdateCRStatusHook{}},
	}
	if err := p.Start(); err != nil {
		if runtime.Arg.InCluster {
			if err := kubekeycontroller.PatchNodeImportStatus(runtime, kubekeycontroller.Failed); err != nil {
				return err
			}
			if err := kubekeycontroller.UpdateStatus(runtime); err != nil {
				return err
			}
		}
		return err
	}

	if runtime.Arg.InCluster {
		if err := kubekeycontroller.PatchNodeImportStatus(runtime, kubekeycontroller.Success); err != nil {
			return err
		}
		if err := kubekeycontroller.UpdateStatus(runtime); err != nil {
			return err
		}
	}

	return nil
}

func NewK3sAddNodesPipeline(runtime *common.KubeRuntime) error {
	noArtifact := runtime.Arg.Artifact == ""

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
		&k3s.JoinNodesModule{},
		&loadbalancer.K3sHaproxyModule{Skip: !runtime.Cluster.ControlPlaneEndpoint.IsInternalLBEnabled()},
		&kubernetes.ConfigureKubernetesModule{},
		&filesystem.ChownModule{},
		&certs.AutoRenewCertsModule{Skip: !runtime.Cluster.Kubernetes.EnableAutoRenewCerts()},
	}

	p := pipeline.Pipeline{
		Name:            "AddNodesPipeline",
		Modules:         m,
		Runtime:         runtime,
		ModulePostHooks: []module.PostHookInterface{&hooks.UpdateCRStatusHook{}},
	}
	if err := p.Start(); err != nil {
		if runtime.Arg.InCluster {
			if err := kubekeycontroller.PatchNodeImportStatus(runtime, kubekeycontroller.Failed); err != nil {
				return err
			}
			if err := kubekeycontroller.UpdateStatus(runtime); err != nil {
				return err
			}
		}
		return err
	}

	if runtime.Arg.InCluster {
		if err := kubekeycontroller.PatchNodeImportStatus(runtime, kubekeycontroller.Success); err != nil {
			return err
		}
		if err := kubekeycontroller.UpdateStatus(runtime); err != nil {
			return err
		}
	}

	return nil
}

func AddNodes(args common.Argument, downloadCmd string) error {
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
		loaderType = common.AllInOne
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
	}

	if runtime.Arg.InCluster {
		if err := kubekeycontroller.CreateNodeForCluster(runtime); err != nil {
			return err
		}
		if err := kubekeycontroller.ClearConditions(runtime); err != nil {
			return err
		}
	}

	switch runtime.Cluster.Kubernetes.Type {
	case common.K3s:
		if err := NewK3sAddNodesPipeline(runtime); err != nil {
			return err
		}
	case common.Kubernetes:
		if err := NewAddNodesPipeline(runtime); err != nil {
			return err
		}
	default:
		if err := NewAddNodesPipeline(runtime); err != nil {
			return err
		}
	}
	return nil
}
