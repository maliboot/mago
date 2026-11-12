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

	t.jwt.SetTimeout(expire)
	t.jwt.SetMaxRefresh(expire)

	return t.jwt.TokenGenerator(map[string]interface{}{"id": subject})
}
