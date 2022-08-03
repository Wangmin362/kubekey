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

package pipeline

import (
	"fmt"
	"os"
	"sync"

	"github.com/pkg/errors"

	"github.com/kubesphere/kubekey/pkg/core/cache"
	"github.com/kubesphere/kubekey/pkg/core/connector"
	"github.com/kubesphere/kubekey/pkg/core/ending"
	"github.com/kubesphere/kubekey/pkg/core/logger"
	"github.com/kubesphere/kubekey/pkg/core/module"
)

var logo = `

 _   __      _          _   __           
| | / /     | |        | | / /           
| |/ / _   _| |__   ___| |/ /  ___ _   _ 
|    \| | | | '_ \ / _ \    \ / _ \ | | |
| |\  \ |_| | |_) |  __/ |\  \  __/ |_| |
\_| \_/\__,_|_.__/ \___\_| \_/\___|\__, |
                                    __/ |
                                   |___/

`

// Pipeline 管道可以理解为一个流水线
type Pipeline struct {
	Name            string            // 流水线的名字
	Modules         []module.Module   // 流水线需要执行的模块
	Runtime         connector.Runtime // 运行时的环境，实际上可以理解为需要安装的几台机器的SSH连接
	SpecHosts       int               // 主机的数量
	PipelineCache   *cache.Cache
	ModuleCachePool sync.Pool
	ModulePostHooks []module.PostHookInterface
}

func (p *Pipeline) Init() error {
	fmt.Print(logo)
	p.PipelineCache = cache.NewCache()
	p.SpecHosts = len(p.Runtime.GetAllHosts())
	//if err := p.Runtime.GenerateWorkDir(); err != nil {
	//	return err
	//}
	//if err := p.Runtime.InitLogger(); err != nil {
	//	return err
	//}
	return nil
}

func (p *Pipeline) Start() error {
	if err := p.Init(); err != nil {
		return errors.Wrapf(err, "Pipeline[%s] execute failed", p.Name)
	}

	// 依次执行每个模块，
	for i := range p.Modules {
		m := p.Modules[i]
		if m.IsSkip() {
			continue
		}

		moduleCache := p.newModuleCache()
		// 从这里可以看出，多个Module共享Pipeline的cache，并且每个module还有自己单独的cache
		m.Default(p.Runtime, p.PipelineCache, moduleCache)
		m.AutoAssert()
		m.Init()
		for j := range p.ModulePostHooks {
			m.AppendPostHook(p.ModulePostHooks[j])
		}

		res := p.RunModule(m)
		err := m.CallPostHook(res)
		if res.IsFailed() {
			return errors.Wrapf(res.CombineResult, "Pipeline[%s] execute failed", p.Name)
		}
		if err != nil {
			return errors.Wrapf(err, "Pipeline[%s] execute failed", p.Name)
		}
		p.releaseModuleCache(moduleCache)
	}
	p.releasePipelineCache()

	// close ssh connect
	for _, host := range p.Runtime.GetAllHosts() {
		p.Runtime.GetConnector().Close(host)
	}

	if p.SpecHosts != len(p.Runtime.GetAllHosts()) {
		return errors.Errorf("Pipeline[%s] execute failed: there are some error in your spec hosts", p.Name)
	}
	logger.Log.Infof("Pipeline[%s] execute successfully", p.Name)
	return nil
}

func (p *Pipeline) RunModule(m module.Module) *ending.ModuleResult {
	m.Slogan()

	result := ending.NewModuleResult()
	for {
		switch m.Is() {
		case module.TaskModuleType:
			m.Run(result)
			if result.IsFailed() {
				return result
			}

		case module.GoroutineModuleType:
			go func() {
				m.Run(result)
				if result.IsFailed() {
					os.Exit(1)
				}
			}()
		default:
			m.Run(result)
			if result.IsFailed() {
				return result
			}
		}

		stop, err := m.Until()
		if err != nil {
			result.LocalErrResult(err)
			return result
		}
		if stop == nil || *stop == true {
			break
		}
	}
	return result
}

func (p *Pipeline) newModuleCache() *cache.Cache {
	moduleCache, ok := p.ModuleCachePool.Get().(*cache.Cache)
	if ok {
		return moduleCache
	}
	return cache.NewCache()
}

func (p *Pipeline) releasePipelineCache() {
	p.PipelineCache.Clean()
}

func (p *Pipeline) releaseModuleCache(c *cache.Cache) {
	c.Clean()
	p.ModuleCachePool.Put(c)
}
