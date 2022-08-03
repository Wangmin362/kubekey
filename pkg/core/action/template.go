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

package action

import (
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/kubesphere/kubekey/pkg/core/connector"
	"github.com/kubesphere/kubekey/pkg/core/util"
	"github.com/pkg/errors"
)

type Template struct {
	BaseAction
	Template *template.Template // 模板渲染
	Dst      string
	Data     util.Data
}

func (t *Template) Execute(runtime connector.Runtime) error {
	templateStr, err := util.Render(t.Template, t.Data)
	if err != nil {
		return errors.Wrap(errors.WithStack(err), fmt.Sprintf("render template %s failed", t.Template.Name()))
	}

	// 把渲染好的文件写入到文件当中
	fileName := filepath.Join(runtime.GetHostWorkDir(), t.Template.Name())
	if err := util.WriteFile(fileName, []byte(templateStr)); err != nil {
		return errors.Wrap(errors.WithStack(err), fmt.Sprintf("write file %s failed", fileName))
	}

	// 拷贝到远程主机当中，远程主机的ssh在运行时环境当中
	if err := runtime.GetRunner().SudoScp(fileName, t.Dst); err != nil {
		return errors.Wrap(errors.WithStack(err), fmt.Sprintf("scp file %s to remote %s failed", fileName, t.Dst))
	}

	return nil
}
