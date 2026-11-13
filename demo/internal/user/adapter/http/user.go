package http

import (
	"context"
	"net/http"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/maliboot/mago"
	"github.com/maliboot/mago/config"

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
// 两种登录都交给 JWTConf 登录中间件：凭证校验在 application，令牌签发 / 有效期 / 成功响应 / 401 包装统一在框架（§12.5）。
type UserController struct {
	mago.AbstractAdapter `wire:"-"`
	service              application.UserService
	// passwordLogin 账号密码登录入口；有效期由回调按请求返回（支持「延长登录」）。
	passwordLogin app.HandlerFunc
	// emailCode 邮箱验证码登录入口。
	emailCode app.HandlerFunc
}

// 非单例：入口适配器，由组合根装配一次。
func NewUserController(service application.UserService, jwt *config.JWTConf) *UserController {
	h := &UserController{service: service}
	if jwt != nil {
		// 不写死 TTL：回调返回 config.LoginResult，按请求决定本次令牌时长。
		h.passwordLogin = jwt.LoginHandlerWith(h.loginByPassword)
		h.emailCode = jwt.LoginHandlerWith(h.loginByEmailCode)
	}

	return h
}

// Routes 返回本上下文注册的对外路由。
// 资源路由按 §7.5 的形状声明；动作类接口必须先登记在 client/api.Actions。
func (h *UserController) Routes() []Route {
	return []Route{
		{Method: http.MethodPost, Path: "/users", Handler: h.Register},
		{Method: http.MethodPost, Path: "/sessions", Handler: h.Login},
		{Method: http.MethodPost, Path: "/sessions/emailCode", Handler: h.EmailCodeLogin},
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

// Login 账号密码登录：凭证校验走 application.LoginByPassword，令牌由 JWT 登录中间件签发（§12.5）。
func (h *UserController) Login(c context.Context, ctx *app.RequestContext) {
	if h.passwordLogin == nil {
		h.Response(ctx, mago.NewVO[mago.EmptyVO](mago.NewError(mago.ErrServerError), nil))
		return
	}

	h.passwordLogin(c, ctx)
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

// EmailCodeLogin 邮箱 + 验证码登录：凭证校验走 application，令牌签发与响应交 JWT 中间件（§12.5）。
func (h *UserController) EmailCodeLogin(c context.Context, ctx *app.RequestContext) {
	if h.emailCode == nil {
		mago.NewResponse(ctx).Failure(mago.ErrServerError)
		return
	}

	h.emailCode(c, ctx)
}

// loginByPassword 校验账密并把登录主体转成令牌载荷；「延长登录」通过 LoginResult.TTL 按请求生效。
func (h *UserController) loginByPassword(c context.Context, ctx *app.RequestContext) (interface{}, error) {
	var qry dto.UserAuthRequest
	if err := ctx.BindAndValidate(&qry); err != nil {
		return nil, mago.NewError(mago.ErrInvalidParams).WithMsg("请检查请求 json 格式、字段类型是否正确")
	}

	// 延长登录：X-Iee: true 时使用 ExtendTokenTTL（§7.5）。
	if string(ctx.GetHeader("X-Iee")) == "true" {
		qry.IsExtendExpire = true
	}

	res := h.service.LoginByPassword(c, &qry)
	if err := res.Error(); err != nil {
		return nil, err
	}

	ttl := application.DefaultTokenTTL
	if qry.IsExtendExpire {
		ttl = application.ExtendTokenTTL
	}

	return config.LoginResult{Claims: subjectClaims(res.Data), TTL: ttl}, nil
}

// loginByEmailCode 把登录主体转成令牌载荷；载荷规则（透传）、有效期与成功响应都在中间件侧。
func (h *UserController) loginByEmailCode(c context.Context, ctx *app.RequestContext) (interface{}, error) {
	var qry dto.EmailCodeLoginRequest
	if err := ctx.BindAndValidate(&qry); err != nil {
		return nil, mago.NewError(mago.ErrInvalidParams).WithMsg("请检查请求 json 格式、字段类型是否正确")
	}

	res := h.service.LoginByEmailCode(c, &qry)
	if err := res.Error(); err != nil {
		return nil, err
	}

	return config.LoginResult{Claims: subjectClaims(res.Data), TTL: application.DefaultTokenTTL}, nil
}

// subjectClaims 是两种登录共用的载荷组装：身份键 id 必须与 JWTAuth.IdentityKey() 一致。
// 框架把这份 map 整体透传进 claims，业务字段（sub/ver 等）按需增删。
func subjectClaims(subject *vo.LoginSubjectVO) map[string]interface{} {
	return map[string]interface{}{
		"id":  strconv.FormatUint(uint64(subject.UserID), 10),
		"sub": subject.Email,
		"ver": subject.TokenVersion,
	}
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
