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
	"fmt"

	"github.com/kubesphere/kubekey/cmd/ctl/options"
	"github.com/kubesphere/kubekey/cmd/ctl/util"
	"github.com/kubesphere/kubekey/pkg/common"
	"github.com/kubesphere/kubekey/pkg/pipelines"
	"github.com/spf13/cobra"
)

type ArtifactExportOptions struct {
	CommonOptions *options.CommonOptions

	ManifestFile string
	Output       string
	CriSocket    string
	DownloadCmd  string
}

func NewArtifactExportOptions() *ArtifactExportOptions {
	return &ArtifactExportOptions{
		CommonOptions: options.NewCommonOptions(),
	}
}

// NewCmdArtifactExport creates a new `kubekey artifact export` command
func NewCmdArtifactExport() *cobra.Command {
	o := NewArtifactExportOptions()
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export a KubeKey offline installation package",
		Run: func(cmd *cobra.Command, args []string) {
			util.CheckErr(o.Complete(cmd, args))
			util.CheckErr(o.Validate(args))
			util.CheckErr(o.Run())
		},
	}

	o.CommonOptions.AddCommonFlag(cmd)
	o.AddFlags(cmd)
	return cmd
}

// Complete 主要是用于补充缺省的参数
func (o *ArtifactExportOptions) Complete(_ *cobra.Command, _ []string) error {
	if o.Output == "" {
		o.Output = "kubekey-artifact.tar.gz"
	}
	return nil
}

func (o *ArtifactExportOptions) Validate(_ []string) error {
	if o.ManifestFile == "" {
		return fmt.Errorf("--manifest can not be an empty string")
	}
	return nil
}

func (o *ArtifactExportOptions) Run() error {
	arg := common.ArtifactArgument{
		ManifestFile: o.ManifestFile, // 打包制品的清单文件
		Output:       o.Output,       // 离线安装包
		CriSocket:    o.CriSocket,
		Debug:        o.CommonOptions.Verbose,
		IgnoreErr:    o.CommonOptions.IgnoreErr,
	}

	// 开启一个pipeline，用于制作离线安装包
	return pipelines.ArtifactExport(arg, o.DownloadCmd)
}

func (o *ArtifactExportOptions) AddFlags(cmd *cobra.Command) {
	// 用于指定资源清单文件
	cmd.Flags().StringVarP(&o.ManifestFile, "manifest", "m", "", "Path to a manifest file")
	cmd.Flags().StringVarP(&o.Output, "output", "o", "", "Path to a output path")
	// todo 下载软件包的命令格式，估计这里一般很难去修改，如果想要加速应下载，应该怎么设置？
	cmd.Flags().StringVarP(&o.DownloadCmd, "download-cmd", "", "curl -L -o %s %s",
		`The user defined command to download the necessary binary files. The first param '%s' is output path, the second param '%s', is the URL`)
}
