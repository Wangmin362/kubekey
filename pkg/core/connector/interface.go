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

package connector

import (
	"io"
	"os"

	"github.com/kubesphere/kubekey/pkg/core/cache"
)

// Connection  这里的一个Connection是一个SSH链接，可以执行命令，实际上目前也只有SSH这么一种实现
type Connection interface {
	// fixme 为什么这里不增加一个 GetHost() (Host, error)方法，这样后面所有的方法就不需要再带上host参数了，可以简便很多啊

	Exec(cmd string, host Host) (stdout string, code int, err error)
	// TODO 这两个方法是咋用的？
	PExec(cmd string, stdin io.Reader, stdout io.Writer, stderr io.Writer, host Host) (code int, err error)
	// TODO 把文件从remote拷贝到local?, host势必是remote的host
	Fetch(local, remote string, host Host) error
	Scp(local, remote string, host Host) error
	RemoteFileExist(remote string, host Host) bool
	RemoteDirExist(remote string, host Host) (bool, error)
	MkDirAll(path string, mode string, host Host) error
	Chmod(path string, mode os.FileMode) error
	// fixme 这个方法好不严谨哦，返回参数应该加上error会好一些吧，链接不一定能正常关闭吧
	Close()
}

type Connector interface {
	Connect(host Host) (Connection, error)
	Close(host Host)
}

type ModuleRuntime interface {
	// TODO 如何理解Obj?
	GetObjName() string
	SetObjName(name string)
	GenerateWorkDir() error
	// TODO 和GetWorkDir方法有啥区别？
	GetHostWorkDir() string
	GetWorkDir() string
	GetIgnoreErr() bool
	GetAllHosts() []Host
	SetAllHosts([]Host)
	GetHostsByRole(role string) []Host
	DeleteHost(host Host)
	HostIsDeprecated(host Host) bool
	InitLogger() error
}

type Runtime interface {
	// TODO 既然Runner中已经有了Connction了，为什么还需要Get/SetConector?
	GetRunner() *Runner
	SetRunner(r *Runner)
	GetConnector() Connector
	SetConnector(c Connector)
	RemoteHost() Host
	Copy() Runtime
	ModuleRuntime
}

// Host 这个应该是对于manifest.yaml中host的一个
type Host interface {
	GetName() string
	SetName(name string)
	GetAddress() string
	SetAddress(str string)
	GetInternalAddress() string
	SetInternalAddress(str string)
	GetPort() int
	SetPort(port int)
	GetUser() string
	SetUser(u string)
	GetPassword() string
	SetPassword(password string)
	GetPrivateKey() string
	SetPrivateKey(privateKey string)
	GetPrivateKeyPath() string
	SetPrivateKeyPath(path string)
	GetArch() string
	SetArch(arch string)
	GetTimeout() int64
	SetTimeout(timeout int64)
	GetRoles() []string
	SetRoles(roles []string)
	IsRole(role string) bool
	GetCache() *cache.Cache
	SetCache(c *cache.Cache)
}
