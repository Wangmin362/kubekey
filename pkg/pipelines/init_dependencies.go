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
	"github.com/kubesphere/kubekey/pkg/artifact"
	"github.com/kubesphere/kubekey/pkg/bootstrap/os"
	"github.com/kubesphere/kubekey/pkg/bootstrap/precheck"
	"github.com/kubesphere/kubekey/pkg/common"
	"github.com/kubesphere/kubekey/pkg/core/module"
	"github.com/kubesphere/kubekey/pkg/core/pipeline"
	"github.com/kubesphere/kubekey/pkg/filesystem"
)

func NewInitDependenciesPipeline(runtime *common.KubeRuntime) error {
	noArtifact := runtime.Arg.Artifact == ""

	m := []module.Module{
		&precheck.GreetingsModule{},                   // 打印kubekey标志语句
		&artifact.UnArchiveModule{Skip: noArtifact},   // 解压缩离线安装包
		&os.RepositoryModule{Skip: noArtifact},        // 离线安装：挂载iso文件，安装必要的软件，然后卸载
		&os.RepositoryOnlineModule{Skip: !noArtifact}, // 在线安装：安装依赖的软件，比如socat, conntrack
		&filesystem.ChownWorkDirModule{},              // 修改.kubekey的 uid gid
	}

	p := pipeline.Pipeline{
		Name:    "InitDependenciesPipeline",
		Modules: m,
		Runtime: runtime,
	}
	if err := p.Start(); err != nil {
		return err
	}
	return nil
}

func InitDependencies(args common.Argument) error {
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

	if err := NewInitDependenciesPipeline(runtime); err != nil {
		return err
	}
	return nil
}
