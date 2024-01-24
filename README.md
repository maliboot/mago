# Mago

#### 简介
* 这是一个`maliboot`框架的golang版本。
* 为了满足`phper`的使用习惯，组件设计、注解(基于google/wire)尽可能的使用了`hyperf`的许多规范。
* 路由组件使用的是`hertz`，数据库组件使用了`gorm`

#### 准备
* golang >= 1.21
* make

#### 安装
```shell
go install github.com/maliboot/mago/mali@latest
```

#### 创建新项目(模块)
* 初始化项目
```shell
mkdir mago-skeleton
cd mago-skeleton
go mod init
```
* 初始化`maliboot`骨架
```shell
cd mago-skeleton
mali init
```

* 生成文件如下
```
.
├── Makefile
├── README.md
├── conf.yml
├── config
│   ├── autoload
│   ├── config.go
│   └── server.go
├── go.mod
├── internal
│   ├── adapter
│   ├── app
│   ├── client
│   ├── domain
│   └── infra
├── main.go
└── wire.go

```

#### 批量生成`CURD`代码
* 修改数据库配置 `mago-skeleton/conf.yml`
```yaml
app_name: example
app_env: dev
server:
  http:
    port: 9501
logger:
  log_dir:
databases:
  ## example为数据库名称
  example:
    dsn: root:root@tcp(127.0.0.1:3306)/example?&parseTime=true&loc=Local
    singular_table: true
redis:
  host: 127.0.0.1
  port: 6379
  db: 1
```

* 使用`cli`工具批量生成`cola`代码
```shell
cd mago-skeleton
## example为数据库名称，uss_message_tpl_var为表名。当无数据库名称时，会默认取`mago-skeleton/conf.yml`里第一个数据库
mali curd example.uss_message_tpl_var
```

> 注意：当使用了注解路由时，需要在`mago-skeleton/main.go`中解开`*Container`注释。另外，`mali curd`默认生成的是注解路由
``` go
package main

import (
    "flag"
    
    "github.com/maliboot/mago"
    "github.com/maliboot/mago/config"
)

var (
	flagConf string
)

func init() {
	flag.StringVar(&flagConf, "f", "./conf.yml", "config path, eg: -f conf.yml")
}

func NewApp(
	c *config.Conf,
	hs *mago.Http,
	// container *Container, // ====================================使用注解路由时需要解开注释，否则编译报错
) *mago.App {
	// inject
	// container.Inject(hs) // =====================================使用注解路由时需要解开注释，否则编译报错

	// app
	return mago.New(
		c.AppName,
		[]mago.Server{hs},
	)
}

func main() {
	flag.Parse()

	// 配置
	c := config.NewConf(config.WithConfFile(flagConf))
	if err := c.Bootstrap(); err != nil {
		panic(err)
	}

	// start
	if err := initApp(c).Run(); err != nil {
		panic(err)
	}
}
```

#### 依赖注入
* 本框架使用了`google/wire`来进行代码依赖注入
* 本框架的注解功能依赖于`google/wire`组件

项目运行前，需要在当前项目下运行依赖注入命令
```shell
cd mago-skeleton
make wire
```

#### 使用
* 运行服务
```
cd mago-skeleton
go run main

....
17:56:51.060784 engine.go:668: [Debug] HERTZ: Method=GET    absolutePath=/ping                     --> handlerName=agentserver/config.NewHttpServer.func1 (num=2 handlers)
17:56:51.060973 engine.go:668: [Debug] HERTZ: Method=POST   absolutePath=/ussMessageTplVar/create  --> handlerName=agentserver/internal/adapter/admin.(*UssMessageTplVarController).Create-fm (num=2 handlers)
17:56:51.060995 engine.go:668: [Debug] HERTZ: Method=DELETE absolutePath=/ussMessageTplVar/delete  --> handlerName=agentserver/internal/adapter/admin.(*UssMessageTplVarController).Delete-fm (num=2 handlers)
17:56:51.061004 engine.go:668: [Debug] HERTZ: Method=GET    absolutePath=/ussMessageTplVar/getById --> handlerName=agentserver/internal/adapter/admin.(*UssMessageTplVarController).GetById-fm (num=2 handlers)
17:56:51.061015 engine.go:668: [Debug] HERTZ: Method=GET    absolutePath=/ussMessageTplVar/listByPage --> handlerName=agentserver/internal/adapter/admin.(*UssMessageTplVarController).ListByPage-fm (num=2 handlers)
17:56:51.061160 engine.go:668: [Debug] HERTZ: Method=PUT    absolutePath=/ussMessageTplVar/update  --> handlerName=agentserver/internal/adapter/admin.(*UssMessageTplVarController).Update-fm (num=2 handlers)
17:56:51.062232 engine.go:396: [Info] HERTZ: Using network library=netpoll
17:56:51.062533 transport.go:115: [Info] HERTZ: HTTP server listening on address=[::]:9501
```

* 浏览器访问 `http://localhost:9501/ping`