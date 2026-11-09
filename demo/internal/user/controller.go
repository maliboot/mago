package user

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/maliboot/mago"
	"github.com/maliboot/mago/config"
)

type Controller struct {
	mago.AbstractAdapter `wire:"-"`
	service              *Service
}

func NewController(c *config.Conf) *Controller {
	return &Controller{
		service: NewService(c),
	}
}

func (h *Controller) Router(routerGroup route.IRoutes, authed route.IRoutes) {
	authed.POST("/auth/login", h.Login)
	routerGroup.POST("/users", h.Register)
}

func (h *Controller) Login(_ context.Context, ctx *app.RequestContext) {
	var qry MemberTokenRequest
	err := ctx.Bind(&qry)
	if err != nil {
		h.Response(ctx, qry.BindErr())
		return
	}

	if string(ctx.GetHeader("X-Iee")) == "true" {
		qry.IsExtendExpire = true
	}

	res := h.service.Login(&qry)
	h.Response(ctx, res)
}

// Register 注册账号
func (h *Controller) Register(_ context.Context, ctx *app.RequestContext) {
	var qry MemberTokenRequest
	err := ctx.Bind(&qry)
	if err != nil {
		h.Response(ctx, qry.BindErr())
		return
	}

	res := h.service.Register(&qry)
	h.Response(ctx, res)
}

func (h *Controller) Me(_ context.Context, ctx *app.RequestContext) {
	var qry MemberTokenRequest
	err := ctx.Bind(&qry)
	if err != nil {
		h.Response(ctx, qry.BindErr())
		return
	}

	res := h.service.Me(&qry)
	h.Response(ctx, res)
}
