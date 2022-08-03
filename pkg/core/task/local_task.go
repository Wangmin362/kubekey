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
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/kubesphere/kubekey/pkg/core/action"
	"github.com/kubesphere/kubekey/pkg/core/cache"
	"github.com/kubesphere/kubekey/pkg/core/common"
	"github.com/kubesphere/kubekey/pkg/core/connector"
	"github.com/kubesphere/kubekey/pkg/core/ending"
	"github.com/kubesphere/kubekey/pkg/core/logger"
	"github.com/kubesphere/kubekey/pkg/core/prepare"
	"github.com/kubesphere/kubekey/pkg/core/rollback"
	"github.com/kubesphere/kubekey/pkg/core/util"
)

// 本地任务的执行，可以理解为在运行kk二进制文件那台机器上要执行的任务，譬如：离线安装包值作的相关任务
type LocalTask struct {
	Name     string
	Desc     string
	Prepare  prepare.Prepare   // 执行任务的前置检查
	Action   action.Action     // 一个任务一个Action, 和kubekey的架构图对应上了
	Rollback rollback.Rollback // fixme 这里应该是action对应的回滚Action，实际上和Action没啥区别，我觉的action.Action和rollback. Rollback接口没啥区别，明显可以统一使用rollback.Rollback接口，没道理需要用两套非常相似的接口啊
	Retry    int               // 任务执行失败的时候，重新执行次数
	Delay    time.Duration     // 重新执行任务时的延时时间，总不能失败了就立马执行吧
	Timeout  time.Duration     // 超时时间，常规参数，没毛病

	PipelineCache *cache.Cache
	ModuleCache   *cache.Cache
	Runtime       connector.Runtime
	tag           string // todo 这个tag咋用的，感觉整体架构和ansible的架构越来越相了，不知道语义是否和ansible中的一致
	IgnoreError   bool
	TaskResult    *ending.TaskResult // action的执行结果
}

func (l *LocalTask) GetDesc() string {
	return l.Desc
}

func (l *LocalTask) Init(runtime connector.Runtime, moduleCache *cache.Cache, pipelineCache *cache.Cache) {
	l.ModuleCache = moduleCache
	l.PipelineCache = pipelineCache
	l.Runtime = runtime
	l.Default()
}

func (l *LocalTask) Default() {
	l.TaskResult = ending.NewTaskResult()
	if l.Name == "" {
		l.Name = DefaultTaskName
	}

	if l.Prepare == nil {
		l.Prepare = new(prepare.BasePrepare)
	}

	if l.Action == nil {
		l.TaskResult.AppendErr(nil, errors.New("the action is nil"))
		return
	}

	if l.Retry <= 0 {
		l.Retry = 1
	}

	if l.Delay <= 0 {
		l.Delay = 5 * time.Second
	}

	if l.Timeout <= 0 {
		// 默认的超时时间为两个小时
		l.Timeout = DefaultTimeout * time.Minute
	}
}

func (l *LocalTask) Execute() *ending.TaskResult {
	if l.TaskResult.IsFailed() {
		return l.TaskResult
	}

	host := &connector.BaseHost{
		Name: common.LocalHost,
	}

	// todo w为啥这里在执行前需要拷贝一次运行时候环境？任务在执行的时候难道还有修改运行时环境的需求？
	selfRuntime := l.Runtime.Copy()
	l.RunWithTimeout(selfRuntime, host)

	if l.TaskResult.IsFailed() {
		l.TaskResult.ErrResult()
		return l.TaskResult
	}

	l.TaskResult.NormalResult()
	return l.TaskResult
}

func (l *LocalTask) RunWithTimeout(runtime connector.Runtime, host connector.Host) {
	ctx, cancel := context.WithTimeout(context.Background(), l.Timeout)
	defer cancel()

	resCh := make(chan error)

	go l.Run(runtime, host, resCh)
	select {
	case <-ctx.Done():
		l.TaskResult.AppendErr(host, fmt.Errorf("execute task timeout, Timeout=%s", util.ShortDur(l.Timeout)))
	case e := <-resCh:
		if e != nil {
			l.TaskResult.AppendErr(host, e)
		}
	}
}

// Run Task的Run方法肯定是在执行Action撒
func (l *LocalTask) Run(runtime connector.Runtime, host connector.Host, resCh chan error) {
	var res error
	defer func() {
		resCh <- res
		close(resCh)
	}()

	runtime.SetRunner(&connector.Runner{
		Conn: nil,
		//Debug: runtime.Arg.Debug,
		Host: host,
	})

	l.Prepare.Init(l.ModuleCache, l.PipelineCache)      // 初始化prepare,也就是前置检查的准备工作
	l.Prepare.AutoAssert(runtime)                       // 断言，看看具体后面具体的任务时怎么执行的
	if ok, err := l.WhenWithRetry(runtime, host); !ok { // prepare的检测执行
		if err != nil {
			res = err
			return
		} else {
			l.TaskResult.AppendSkip(host)
			return
		}
	}

	l.Action.Init(l.ModuleCache, l.PipelineCache)             // action的初始化
	l.Action.AutoAssert(runtime)                              // action的自动断言
	if err := l.ExecuteWithRetry(runtime, host); err != nil { // 执行实际的动作，Task实际起作用就是从这里开始的
		res = err
		return
	}
	l.TaskResult.AppendSuccess(host)
}

func (l *LocalTask) WhenWithRetry(runtime connector.Runtime, host connector.Host) (bool, error) {
	pass := false
	err := fmt.Errorf("pre-check exec failed after %d retires", l.Retry)
	for i := 0; i < l.Retry; i++ {
		if res, e := l.When(runtime); e != nil {
			logger.Log.Messagef(host.GetName(), e.Error())

			if i == l.Retry-1 {
				err = errors.New(err.Error() + e.Error())
				continue
			}
			logger.Log.Infof("retry: [%s]", host.GetName())
			time.Sleep(l.Delay)
			continue
		} else { // 只有前置检查出现错误了才需要重新执行
			err = nil
			pass = res
			break
		}
	}

	return pass, err
}

func (l *LocalTask) When(runtime connector.Runtime) (bool, error) {
	if l.Prepare == nil { // 没有前置检查就认为可以执行
		return true, nil
	}

	// 前置检测
	if ok, err := l.Prepare.PreCheck(runtime); err != nil {
		return false, err
	} else if !ok {
		return false, nil
	}
	return true, nil
}

func (l *LocalTask) ExecuteWithRetry(runtime connector.Runtime, host connector.Host) error {
	err := fmt.Errorf("[%s] exec failed after %d retires: ", l.Name, l.Retry)
	for i := 0; i < l.Retry; i++ {
		e := l.Action.Execute(runtime) // 执行action
		if e != nil {
			logger.Log.Messagef(host.GetName(), e.Error())

			if i == l.Retry-1 {
				err = errors.New(err.Error() + e.Error())
				continue
			}
			logger.Log.Infof("retry: [%s]", host.GetName())
			time.Sleep(l.Delay)
			continue
		} else {
			err = nil
			break
		}
	}
	return err
}

func (l *LocalTask) ExecuteRollback() {
	if l.Rollback == nil {
		return
	}
	if !l.TaskResult.IsFailed() {
		return
	}
}
