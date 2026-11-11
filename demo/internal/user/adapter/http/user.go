package http

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/maliboot/mago"

	"tests/internal/user/application"
	"tests/internal/user/client/dto"
)

// UserController 是用户上下文的 HTTP 入口，只做协议转换，业务规则在 application / domain。
type UserController struct {
	mago.AbstractAdapter `wire:"-"`
	service              application.UserService
}

func NewUserController(service application.UserService) *UserController {
	return &UserController{
		service: service,
	}
}

func (h *UserController) Router(routerGroup route.IRoutes, authed route.IRoutes) {
	authed.POST("/auth/login", h.Login)
	routerGroup.POST("/users", h.Register)
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

func (h *UserController) Me(c context.Context, ctx *app.RequestContext) {
	var qry dto.UserAuthRequest
	err := ctx.Bind(&qry)
	if err != nil {
		h.Response(ctx, qry.BindErr())
		return
	}

	res := h.service.Me(c, qry.UserId)
	h.Response(ctx, res)
}
