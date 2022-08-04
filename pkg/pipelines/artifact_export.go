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
	"github.com/kubesphere/kubekey/pkg/binaries"
	"github.com/kubesphere/kubekey/pkg/bootstrap/confirm"
	"github.com/kubesphere/kubekey/pkg/common"
	"github.com/kubesphere/kubekey/pkg/core/module"
	"github.com/kubesphere/kubekey/pkg/core/pipeline"
	"github.com/kubesphere/kubekey/pkg/filesystem"
	"github.com/kubesphere/kubekey/pkg/images"
)

func NewArtifactExportPipeline(runtime *common.ArtifactRuntime) error {
	m := []module.Module{
		&confirm.CheckFileExistModule{FileName: runtime.Arg.Output}, // 检查即将要生成的离线部署包名字是否已经存在，如果已经存在，用户必须同意kk覆盖该文件，否则直接退出程序
		&images.CopyImagesToLocalModule{},                           // 下载镜像，保存的方式应该就是遵守了OCI的方式
		&binaries.ArtifactBinariesModule{},                          // 下载二进制安装文件，譬如etcd, kubeadm, kubelet, kubectl, kubecni, helm, crictl, runc, containerd
		&artifact.RepositoryModule{},                                // 下载IOS文件
		&artifact.ArchiveModule{},                                   // 文件归档，实际上可以理解为把之前所有下载好的文件压缩为 kubekey-artifact.tar.gz文件
		&filesystem.ChownOutputModule{},                             // 利用chmod命令，改变文件的 uid gid
		&filesystem.ChownWorkDirModule{},                            // 利用chmod命令，改变工作目录的权限
	}

	p := pipeline.Pipeline{
		Name:            "ArtifactExportPipeline",
		Modules:         m,
		Runtime:         runtime,
		ModulePostHooks: nil,
	}
	if err := p.Start(); err != nil {
		return err
	}

	return nil
}

func NewK3sArtifactExportPipeline(runtime *common.ArtifactRuntime) error {
	m := []module.Module{
		&confirm.CheckFileExistModule{FileName: runtime.Arg.Output},
		&images.CopyImagesToLocalModule{},
		&binaries.K3sArtifactBinariesModule{},
		&artifact.RepositoryModule{},
		&artifact.ArchiveModule{},
		&filesystem.ChownOutputModule{},
		&filesystem.ChownWorkDirModule{},
	}

	p := pipeline.Pipeline{
		Name:            "K3sArtifactExportPipeline",
		Modules:         m,
		Runtime:         runtime,
		ModulePostHooks: nil,
	}
	if err := p.Start(); err != nil {
		return err
	}

	return nil
}

func ArtifactExport(args common.ArtifactArgument, downloadCmd string) error {
	// 拼接下载软件的URL
	args.DownloadCommand = func(path, url string) string {
		// this is an extension point for downloading tools, for example users can set the timeout, proxy or retry under
		// some poor network environment. Or users even can choose another cli, it might be wget.
		// perhaps we should have a build-in download function instead of totally rely on the external one
		return fmt.Sprintf(downloadCmd, path, url)
	}

	// runtime主要是包含了主机的连接信息，可以通过ssh用来传输命令
	runtime, err := common.NewArtifactRuntime(args)
	if err != nil {
		return err
	}

	// 必须要安装k8s
	if len(runtime.Spec.KubernetesDistributions) == 0 {
		return errors.New("the length of kubernetes distributions can't be 0")
	}

	// 类型必须一致，不能k8s,k3s混着来
	pre := runtime.Spec.KubernetesDistributions[0].Type
	for _, t := range runtime.Spec.KubernetesDistributions {
		if t.Type != pre {
			return errors.New("all the types of kubernetes distributions can't be different")
		}
	}

	switch runtime.Spec.KubernetesDistributions[0].Type {
	case common.K3s:
		if err := NewK3sArtifactExportPipeline(runtime); err != nil {
			return err
		}
	case common.Kubernetes:
		if err := NewArtifactExportPipeline(runtime); err != nil {
			return err
		}
	default:
		if err := NewArtifactExportPipeline(runtime); err != nil {
			return err
		}
	}

	return nil
}
