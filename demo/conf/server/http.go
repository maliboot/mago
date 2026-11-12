package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/middlewares/server/recovery"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/maliboot/mago"
	"github.com/maliboot/mago/autoloadx"
	"github.com/maliboot/mago/config"
	eghelper "github.com/maliboot/mago/helper"
)

// NewHttpServer 只构造 hertz server 与通用中间件（access log、request id、recovery、静态目录、探活），
// 不碰业务路由：adapter 入口由 wire 装配成 conf.Handler，再由 Handler.Bind 挂到本 engine 上。
func NewHttpServer(c *config.Conf) *mago.Http {
	// http
	h := mago.NewHttp(c, server.Default(
		server.WithHostPorts(fmt.Sprintf("%s:%d", c.Server.Http.IP, c.Server.Http.Port)),
		server.WithMaxRequestBodySize(1024*1024*4),
		server.WithCustomBinder(autoloadx.NewXBinder()),
	))

	// 自定义初始化
	mago.ErrNone = 0

	// access log
	h.Use(autoloadx.AccessLog(c))
	// request id
	h.Use(autoloadx.RequestIDNew())
	// recover
	h.Use(recovery.Recovery(recovery.WithRecoveryHandler(func(_ context.Context, ctx *app.RequestContext, err interface{}, stack []byte) {
		hlog.Errorf("panic: %v[%s]", err, eghelper.AesEncryptLog(stack, ""))
		mago.NewResponse(ctx).Failure(mago.ErrServerError)
	})))

	// 手动注册路由
	if c.File.IsFS() {
		fsRoot, _ := c.File.GetFSAbsDir()
		h.StaticFS("/static", &app.FS{Root: fsRoot, PathRewrite: func(ctx *app.RequestContext) []byte {
			if strings.HasPrefix(string(ctx.Path()), "/static") {
				return []byte(strings.TrimPrefix(string(ctx.Path()), "/static"))
			}
			return ctx.Path()
		}})
	}
	h.GET("/ping", func(c context.Context, ctx *app.RequestContext) {
		mago.NewResponse(ctx).Success("pong")
	})

	autoloadx.TenantJwtInit(c, "frontend")
	return h
}
