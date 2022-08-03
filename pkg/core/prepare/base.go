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
	"github.com/kubesphere/kubekey/pkg/core/cache"
	"github.com/kubesphere/kubekey/pkg/core/connector"
)

type BasePrepare struct {
	ModuleCache   *cache.Cache
	PipelineCache *cache.Cache
}

func (b *BasePrepare) Init(moduleCache *cache.Cache, pipelineCache *cache.Cache) {
	b.ModuleCache = moduleCache
	b.PipelineCache = pipelineCache
}

func (b *BasePrepare) PreCheck(runtime connector.Runtime) (bool, error) {
	return true, nil
}

func (b *BasePrepare) AutoAssert(runtime connector.Runtime) {
}

// PrepareCollection 直接命名为：Prepares不就好了, 感觉这个人应该是Java老手
type PrepareCollection []Prepare

func (p *PrepareCollection) Init(cache *cache.Cache, rootCache *cache.Cache) {
	for _, v := range *p {
		v.Init(cache, rootCache)
	}
}

func (p *PrepareCollection) PreCheck(runtime connector.Runtime) (bool, error) {
	for _, v := range *p {
		res, err := v.PreCheck(runtime)
		if err != nil {
			return false, err
		}
		// fixme 这段代码就要吐槽了，一般新手才这么写代码吧，没有codeReview嘛？ bool类型直接使用就好了哇
		if res == false {
			return false, nil
		}
	}
	return true, nil
}

func (p *PrepareCollection) AutoAssert(runtime connector.Runtime) {
	for _, v := range *p {
		v.AutoAssert(runtime)
	}
}
