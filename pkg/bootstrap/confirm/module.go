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

package confirm

import (
	"github.com/kubesphere/kubekey/pkg/common"
	"github.com/kubesphere/kubekey/pkg/core/module"
	"github.com/kubesphere/kubekey/pkg/core/task"
)

type InstallConfirmModule struct {
	common.KubeModule
	Skip bool
}

func (i *InstallConfirmModule) IsSkip() bool {
	return i.Skip
}

func (i *InstallConfirmModule) Init() {
	i.Name = "ConfirmModule"
	i.Desc = "Display confirmation form"

	display := &task.LocalTask{
		Name:   "ConfirmForm",
		Desc:   "Display confirmation form",
		Action: new(InstallationConfirm),
	}

	i.Tasks = []task.Interface{
		display,
	}
}

type DeleteClusterConfirmModule struct {
	common.KubeModule
}

func (d *DeleteClusterConfirmModule) Init() {
	d.Name = "DeleteClusterConfirmModule"
	d.Desc = "Display delete confirmation form"

	display := &task.LocalTask{
		Name:   "ConfirmForm",
		Desc:   "Display confirmation form",
		Action: &DeleteConfirm{Content: "cluster"},
	}

	d.Tasks = []task.Interface{
		display,
	}
}

type DeleteNodeConfirmModule struct {
	common.KubeModule
}

func (d *DeleteNodeConfirmModule) Init() {
	d.Name = "DeleteNodeConfirmModule"
	d.Desc = "Display delete node confirmation form"

	display := &task.LocalTask{
		Name:   "ConfirmForm",
		Desc:   "Display confirmation form",
		Action: &DeleteConfirm{Content: "node"},
	}

	d.Tasks = []task.Interface{
		display,
	}
}

type UpgradeConfirmModule struct {
	common.KubeModule
	Skip bool
}

func (u *UpgradeConfirmModule) IsSkip() bool {
	return u.Skip
}

func (u *UpgradeConfirmModule) Init() {
	u.Name = "UpgradeConfirmModule"
	u.Desc = "Display upgrade confirmation form"

	display := &task.LocalTask{
		Name:   "ConfirmForm",
		Desc:   "Display confirmation form",
		Action: new(UpgradeConfirm),
	}

	u.Tasks = []task.Interface{
		display,
	}
}

type CheckFileExistModule struct {
	module.BaseTaskModule
	FileName string
}

func (c *CheckFileExistModule) Init() {
	c.Name = "CheckFileExist"
	c.Desc = "Check file if is existed"

	check := &task.LocalTask{
		Name: "CheckExist",
		Desc: "Check output file if existed",
		// 这个action仅仅是检测文件是否存在，如果已经存在了，那么用户必须统一覆盖，否则就直接退出了
		Action: &CheckFile{FileName: c.FileName}, // 检测文件是否存在的Action, 由于文件的检测并没有实际做什么操作，所以这里也并没有回滚的回调

	}

	c.Tasks = []task.Interface{
		check,
	}
}
