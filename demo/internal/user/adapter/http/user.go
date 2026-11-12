package http

import (
	"context"
	"net/http"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/maliboot/mago"

	"tests/internal/user/application"
	"tests/internal/user/client/dto"
	"tests/internal/user/client/vo"
)

// Route 描述本控制器的一条路由，集中声明便于契约测试校验路径形状。
type Route struct {
	// Method 是 HTTP 方法。
	Method string
	// Path 是相对上下文分组（/api/v1/user）的路径。
	Path string
	// Authed 为真时挂在需要 JWT 的分组上。
	Authed bool
	// Handler 是处理函数。
	Handler app.HandlerFunc
}

// UserController 是用户上下文的 HTTP 入口，只做协议转换，业务规则在 application / domain。
type UserController struct {
	mago.AbstractAdapter `wire:"-"`
	service              application.UserService
}

// 非单例：入口适配器，由组合根装配一次。
func NewUserController(service application.UserService) *UserController {
	return &UserController{
		service: service,
	}
}

// Routes 返回本上下文注册的对外路由。
// 资源路由按 §7.5 的形状声明；动作类接口必须先登记在 client/api.Actions。
func (h *UserController) Routes() []Route {
	return []Route{
		{Method: http.MethodPost, Path: "/users", Handler: h.Register},
		{Method: http.MethodPost, Path: "/sessions", Handler: h.Login},
		{Method: http.MethodPost, Path: "/users/verify-code", Handler: h.SendVerifyCode},
		{Method: http.MethodGet, Path: "/users/:id", Authed: true, Handler: h.Me},
	}
}

func (h *UserController) Router(routerGroup route.IRoutes, authed route.IRoutes) {
	for _, r := range h.Routes() {
		target := routerGroup
		if r.Authed {
			target = authed
		}

		target.Handle(r.Method, r.Path, r.Handler)
	}
}

func (h *UserController) Login(c context.Context, ctx *app.RequestContext) {
	var qry dto.UserAuthRequest
	err := ctx.Bind(&qry)
	if err != nil {
		h.Response(ctx, qry.BindErr())
		return
	}

	if string(ctx.GetHeader("X-Iee")) == "true" {
		qry.IsExtendExpire = true
	}

	res := h.service.Login(c, &qry)
	h.Response(ctx, res)
}

// Register 注册账号
func (h *UserController) Register(c context.Context, ctx *app.RequestContext) {
	var qry dto.UserAuthRequest
	err := ctx.Bind(&qry)
	if err != nil {
		h.Response(ctx, qry.BindErr())
		return
	}

	res := h.service.Register(c, &qry)
	h.Response(ctx, res)
}

// SendVerifyCode 发送 6 位验证码到邮箱
func (h *UserController) SendVerifyCode(c context.Context, ctx *app.RequestContext) {
	var qry dto.VerifyCodeRequest
	err := ctx.Bind(&qry)
	if err != nil {
		h.Response(ctx, qry.BindErr())
		return
	}

	res := h.service.SendVerifyCode(c, &qry)
	h.Response(ctx, res)
}

// Me 按路径参数中的用户 ID 查询详情
func (h *UserController) Me(c context.Context, ctx *app.RequestContext) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || id == 0 {
		h.Response(ctx, mago.NewVO[vo.UserVO](mago.NewError(mago.ErrInvalidParams).WithMsg("用户ID非法"), nil))
		return
	}

	res := h.service.Me(c, uint(id))
	h.Response(ctx, res)
}
