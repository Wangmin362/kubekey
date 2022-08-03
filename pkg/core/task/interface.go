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

package task

import (
	"github.com/kubesphere/kubekey/pkg/core/cache"
	"github.com/kubesphere/kubekey/pkg/core/connector"
	"github.com/kubesphere/kubekey/pkg/core/ending"
)

// Interface 是对与Task的一个抽象，规定了Task应该有以下四个功能：获取任务的描述信息、初始化Task、
// 执行任务（执行任务需要的环境是通过Init接口注入的，主要注入了连接信息以及模块缓存、pipeline缓存）
// 另外还有一个回滚接口
//
// 在Kubekey中，任务分为两种：本地任务和远程任务，根据我对于kubekey使用的理解，本地任务执行肯定是在执行
// kubekey二进制文件那台机器上执行的，按照我的理解应该是离线安装包的打包相关的任务都是本地任务，
// 而对于部署k8s集群相关的任务，应该就是一个远程任务，因为一般来说一个生产可用的K8S集群应该是要在多个节点上的
// 而部署任务肯定是要通过SSH远程执行的，所以猜测是远程任务。
// 那么执行kk的那台机器，如果本身就是需要部署的一个节点，那么这台机器上的任务是远程任务还是本地任务？
type Interface interface {
	GetDesc() string
	// fixme 这里又来一个init接口，和之前的Default接口的签名完全一样的嘛
	Init(runtime connector.Runtime, moduleCache *cache.Cache, pipelineCache *cache.Cache)
	Execute() *ending.TaskResult
	// todo 回滚功能，模块在实现这个功能的时候怕是有点不好做哦，很多都是shell命令直接执行的，撤销的话没有那么好做，我猜测实际实现的模块没有实现这个接口
	ExecuteRollback()
}
