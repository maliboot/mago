package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/maliboot/mago"

	"tests/internal/user/application"
	"tests/internal/user/client/enum"
)

// JWTAuth 是本上下文的 JWT 扩展实现：身份键、动态作废规则与登录成功响应字段都在这里。
// 组合根把它装进 config.JWTConf（conf/handler.go 的 Bind），框架只管签发与校验（§12.5）。
type JWTAuth struct {
	service application.UserService
}

// NewJWTAuth 非单例：入口适配器，由组合根装配一次。
func NewJWTAuth(service application.UserService) *JWTAuth {
	return &JWTAuth{service: service}
}

// IdentityKey 身份键用用户 ID，与签发时写入的 claims 一致。
func (a *JWTAuth) IdentityKey() string { return "id" }

// Login 是框架默认账密入口的钩子；本 demo 的两种登录都走 LoginHandlerWith 自定义入口
// （见 user.go 的 loginByPassword / loginByEmailCode），因此这里返回 nil 不启用默认入口。
func (a *JWTAuth) Login(username, password, ext string) map[string]interface{} { return nil }

// Validate 老签名没有 ctx，查库会逼着用 context.Background()，因此改用 ValidateIdentity。
func (a *JWTAuth) Validate() func(identityVal string) bool { return nil }

// ValidateIdentity 演示动态作废：令牌里的用户在库中不存在或已停用时，下一个请求立刻拒绝。
// 真实项目不要每请求查库，用 Redis 黑名单或用户表 token_version 比对。
func (a *JWTAuth) ValidateIdentity(ctx context.Context, identityVal string) bool {
	id, err := strconv.ParseUint(identityVal, 10, 64)
	if err != nil || id == 0 {
		return false
	}

	res := a.service.Me(ctx, uint(id))
	if res.Error() != nil || res.Data == nil {
		return false
	}

	return res.Data.Status == enum.UserStatusActive
}

// ResponseWrapper 非登录类成功响应（如退出登录）走统一响应格式。
func (a *JWTAuth) ResponseWrapper(ctx context.Context, c *app.RequestContext, code int, message map[string]interface{}) {
	mago.NewResponse(c).Success(message)
}

// UnauthorizedWrapper 401/403 统一成 mago 错误码，与其它接口的响应形状一致。
// 403 是「令牌有效但身份被拒」（停用、校验不通过）：对前端统一成「重新登录」，
// 框架的英文细节不回给客户端；401 保留业务文案（如「验证码错误或已过期」）。
func (a *JWTAuth) UnauthorizedWrapper(ctx context.Context, c *app.RequestContext, code int, message string) {
	if code == http.StatusForbidden {
		message = "登录状态已失效，请重新登录"
	}

	mago.NewResponse(c).FailureError(mago.NewError(mago.ErrAuthUnauthorized).WithMsg(message))
}

// LoginResponse 自定义登录成功响应字段：除 token/expire 外带回登录主体（id、邮箱、令牌版本），
// 前端不用再补一次「当前用户」请求。字段按前端契约增删，框架不做限制（§12.5）。
func (a *JWTAuth) LoginResponse(ctx context.Context, c *app.RequestContext, token string, expireAt time.Time, claims map[string]interface{}) {
	mago.NewResponse(c).Success(map[string]interface{}{
		"token":        token,
		"expire":       expireAt.Format(time.DateTime),
		"userId":       claims["id"],
		"email":        claims["sub"],
		"tokenVersion": claims["ver"],
	})
}
