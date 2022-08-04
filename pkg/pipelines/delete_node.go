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
	"github.com/kubesphere/kubekey/pkg/bootstrap/confirm"
	"github.com/kubesphere/kubekey/pkg/bootstrap/os"
	"github.com/kubesphere/kubekey/pkg/bootstrap/precheck"
	"github.com/kubesphere/kubekey/pkg/common"
	"github.com/kubesphere/kubekey/pkg/core/module"
	"github.com/kubesphere/kubekey/pkg/core/pipeline"
	"github.com/kubesphere/kubekey/pkg/kubernetes"
)

func DeleteNodePipeline(runtime *common.KubeRuntime) error {
	m := []module.Module{
		&precheck.GreetingsModule{},                     // 打印kk标志语
		&confirm.DeleteNodeConfirmModule{},              // 跟用户确认是否真的要删除节点
		&kubernetes.CompareConfigAndClusterInfoModule{}, // 获取到需要删除的节点
		&kubernetes.DeleteKubeNodeModule{},              // 1、kubectl drain  2、kubectl delete node
		&os.ClearNodeOSModule{},                         // 重置网络，删除不必要的文件，重启systemd
	}

	p := pipeline.Pipeline{
		Name:    "DeleteNodePipeline",
		Modules: m,
		Runtime: runtime,
	}
	if err := p.Start(); err != nil {
		return err
	}
	return nil
}

func DeleteNode(args common.Argument) error {
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

	if err := DeleteNodePipeline(runtime); err != nil {
		return err
	}
	return nil
}
