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

	"github.com/pkg/errors"

	"github.com/kubesphere/kubekey/pkg/artifact"
	"github.com/kubesphere/kubekey/pkg/bootstrap/confirm"
	"github.com/kubesphere/kubekey/pkg/bootstrap/precheck"
	"github.com/kubesphere/kubekey/pkg/certs"
	"github.com/kubesphere/kubekey/pkg/common"
	"github.com/kubesphere/kubekey/pkg/core/module"
	"github.com/kubesphere/kubekey/pkg/core/pipeline"
	"github.com/kubesphere/kubekey/pkg/filesystem"
	"github.com/kubesphere/kubekey/pkg/kubernetes"
	"github.com/kubesphere/kubekey/pkg/kubesphere"
	"github.com/kubesphere/kubekey/pkg/loadbalancer"
)

func NewUpgradeClusterPipeline(runtime *common.KubeRuntime) error {
	noArtifact := runtime.Arg.Artifact == ""

	m := []module.Module{
		&precheck.GreetingsModule{},                                                                    // 打印kubekey标志语句
		&precheck.NodePreCheckModule{},                                                                 // 检查系统软件的安装情况
		&precheck.ClusterPreCheckModule{},                                                              // 检查当前K8S集群的状态信息
		&confirm.UpgradeConfirmModule{Skip: runtime.Arg.SkipConfirmCheck},                              // 提示用户是否真的需要升级集群
		&artifact.UnArchiveModule{Skip: noArtifact},                                                    // 解压缩离线安装包
		&kubernetes.SetUpgradePlanModule{Step: kubernetes.ToV121},                                      // 升级到 1.21的检测工作
		&kubernetes.ProgressiveUpgradeModule{Step: kubernetes.ToV121},                                  // 准备必要的安装文件
		&loadbalancer.HaproxyModule{Skip: !runtime.Cluster.ControlPlaneEndpoint.IsInternalLBEnabled()}, // 安装Haproxy
		&kubesphere.CleanClusterConfigurationModule{Skip: !runtime.Cluster.KubeSphere.Enabled},         //
		&kubesphere.ConvertModule{Skip: !runtime.Cluster.KubeSphere.Enabled},                           // 把ks-installer的版本从v2切换到v3
		&kubesphere.DeployModule{Skip: !runtime.Cluster.KubeSphere.Enabled},                            // 部署ks-installer
		&kubesphere.CheckResultModule{Skip: !runtime.Cluster.KubeSphere.Enabled},                       // 检查ks-installer的部署情况
		&kubernetes.SetUpgradePlanModule{Step: kubernetes.ToV122},                                      // 升级到1.22的检测工作
		&kubernetes.ProgressiveUpgradeModule{Step: kubernetes.ToV122},
		&filesystem.ChownModule{},
		&certs.AutoRenewCertsModule{Skip: !runtime.Cluster.Kubernetes.EnableAutoRenewCerts()},
	}

	p := pipeline.Pipeline{
		Name:    "UpgradeClusterPipeline",
		Modules: m,
		Runtime: runtime,
	}
	if err := p.Start(); err != nil {
		return err
	}
	return nil
}

func UpgradeCluster(args common.Argument, downloadCmd string) error {
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

	switch runtime.Cluster.Kubernetes.Type {
	case common.Kubernetes:
		if err := NewUpgradeClusterPipeline(runtime); err != nil {
			return err
		}
	default:
		return errors.New("unsupported cluster kubernetes type")
	}

	return nil
}
