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

package module

import (
	"github.com/kubesphere/kubekey/pkg/core/cache"
	"github.com/kubesphere/kubekey/pkg/core/connector"
	"github.com/kubesphere/kubekey/pkg/core/ending"
)

type Module interface {
	IsSkip() bool
	// fixme，感觉这个名字不是很好，也不加入注释，default方法是想要干嘛，完全看不出来啊
	Default(runtime connector.Runtime, pipelineCache *cache.Cache, moduleCache *cache.Cache)
	// 初始化模块
	Init()
	// fixme 这个也完全看不出来，注释也不写，看代码才知道适用于标记当前模块是那种类型，那为啥返回值不用自定义类型啊。。。。。。
	Is() string
	Run(result *ending.ModuleResult)
	Until() (*bool, error)
	// 模块开始执行时打印的标志语？
	Slogan()
	// TODO 自动断言应该怎么使用？ 该方法再什么时候被调用？
	AutoAssert()
	// 下面两个hook点有啥区别？
	AppendPostHook(h PostHookInterface)
	CallPostHook(result *ending.ModuleResult) error
}
