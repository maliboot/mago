package repository

import (
	"context"
	"time"

	"github.com/maliboot/mago"
	"github.com/maliboot/mago/config"
)

// Token 是 domain.TokenIssuer 的 JWT 实现，技术细节（hertz JWT 中间件、配置）都收在这里。
type Token struct {
	jwt *config.JWTConf
}

// 非单例：依赖签发配置，按配置实例化，由组合根创建一次。
func NewToken(c *config.Conf) *Token {
	return &Token{jwt: c.Server.Http.JWT["frontend"]}
}

func (t *Token) Issue(_ context.Context, subject string, expire time.Duration) (string, time.Time, error) {
	if t.jwt == nil {
		return "", time.Time{}, mago.NewError(mago.ErrServerError)
	}

	// 按次指定有效期签发：不同调用方（后台长令牌、移动端短令牌）各用各的 TTL，
	// 不要用 SetTimeout 改共享中间件，那样会改到并发中的其它请求。
	return t.jwt.TokenGeneratorWithTTL(map[string]interface{}{"id": subject}, expire)
}
