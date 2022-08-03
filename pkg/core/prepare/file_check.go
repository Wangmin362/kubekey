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

package prepare

import "github.com/kubesphere/kubekey/pkg/core/connector"

type FileExist struct {
	BasePrepare
	FilePath string
	Not      bool // 加入这个Not变量，应该是想要同时实现判断文件不存在和文件存在的情况，不过参数命名还是感觉有问题，虽然想不到更好的命名，但是这里的命名我是持有质疑的
}

func (f *FileExist) PreCheck(runtime connector.Runtime) (bool, error) {
	exist, err := runtime.GetRunner().FileExist(f.FilePath)
	if err != nil {
		return false, err
	}
	if f.Not {
		return !exist, nil
	}
	return exist, nil
}
