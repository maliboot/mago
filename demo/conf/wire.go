//go:build wireinject
// +build wireinject

package conf

import (
	"github.com/google/wire"
	"github.com/maliboot/mago/config"

	userhttp "tests/internal/user/adapter/http"
	usermcp "tests/internal/user/adapter/mcp"
	"tests/internal/user/application"
	"tests/internal/user/infrastructure/repository"
)

// InitHandler 是组合根唯一的注入入口：wire 按类型把配置与四层实现装配成 Handler。
// 生成命令：go tool wire ./conf（生成物 conf/wire_gen.go 入库，改动后必须重新生成）。
func InitHandler(c *config.Conf) (*Handler, error) {
	wire.Build(
		// 子配置切片：provider 只暴露使用方真正需要的那一段配置
		provideSMTPConf,

		// infrastructure：一个 DO 一个 repository，读写端口都由它实现
		repository.NewUser,
		provideUserGateway,
		provideUserQuery,
		repository.NewPassword,
		repository.NewToken,
		repository.NewSMTP,

		// application：技术端口由使用方定义，实现由 infrastructure 注入
		providePasswordHasher,
		provideTokenIssuer,
		provideCodeSender,
		application.NewUserService,

		// adapter：HTTP 与 MCP 共用同一个 Service 实例
		userhttp.NewUserController,
		usermcp.NewUserMcp,

		// 组合根产物：只装 adapter 入口，Server 由 main 的 NewApp 汇总
		wire.Struct(new(Handler), "*"),
	)

	return nil, nil
}
