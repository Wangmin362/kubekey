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
	"github.com/kubesphere/kubekey/pkg/core/cache"
	"github.com/kubesphere/kubekey/pkg/core/connector"
)

// Action 为Task中具体要执行的一个动作的抽象， todo 吐槽一下，感觉这些接口的抽象好相似哦，有没有一种更好的抽象方式
type Action interface {
	Execute(runtime connector.Runtime) (err error)
	Init(cache *cache.Cache, rootCache *cache.Cache) // 哪里都有你们，有没有一种更好的抽象？
	AutoAssert(runtime connector.Runtime)
}
