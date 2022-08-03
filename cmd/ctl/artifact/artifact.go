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

package artifact

import (
	"github.com/kubesphere/kubekey/cmd/ctl/artifact/images"
	"github.com/kubesphere/kubekey/cmd/ctl/options"
	"github.com/spf13/cobra"
)

type ArtifactOptions struct {
	CommonOptions *options.CommonOptions
}

func NewArtifactOptions() *ArtifactOptions {
	return &ArtifactOptions{
		CommonOptions: options.NewCommonOptions(),
	}
}

// NewCmdArtifact creates a new cobra.Command for `kubekey artifact`
// 该命令用于制作打包制品，也就是打一个离线安装包，离线安装包中主要包含三大部分
// 1、操作系统ISO文件，主要用于在离线的机器上安装必备的软件，譬如socat, conntrack
// 2、组件，其中包括：helm, cni, etcd, container-runtime, crictl, harbor
// 3、安装K8S必备的软件或者感兴趣的镜像，也就是manifest.yaml中的images部分
func NewCmdArtifact() *cobra.Command {
	o := NewArtifactOptions()
	cmd := &cobra.Command{
		Use:   "artifact",
		Short: "Manage a KubeKey offline installation package",
	}

	o.CommonOptions.AddCommonFlag(cmd)

	cmd.AddCommand(NewCmdArtifactExport()) // 制作离线安装包
	cmd.AddCommand(images.NewCmdArtifactImages())
	return cmd
}
