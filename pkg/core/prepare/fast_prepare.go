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

import (
	"github.com/kubesphere/kubekey/pkg/core/connector"
)

// todo 没理解为为啥注入了运行时的环境，就被称为FastPrepare，感觉这个命名还是有待商榷
// 另外，BasePrepare中已经的PreCheck方法已经注入了运行时环境，为什么还需要这个实现，
// 从IDEA的提示看来，这个实现也从来没有被用到过
type FastPrepare struct {
	BasePrepare
	Inject func(runtime connector.Runtime) (bool, error)
}

func (b *FastPrepare) PreCheck(runtime connector.Runtime) (bool, error) {
	return b.Inject(runtime)
}
