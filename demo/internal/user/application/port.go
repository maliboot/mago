package application

import (
	"context"
	"time"
)

// PasswordHasher 是密码散列能力端口，由 infrastructure 实现。
// 使用方是 application（用例），所以按「接口由使用方定义」放在这里，而不是 domain。
type PasswordHasher interface {
	Hash(ctx context.Context, raw string) (string, error)
	Verify(ctx context.Context, raw, hash string) bool
}

// TokenIssuer 是访问令牌签发能力端口，由 infrastructure 实现。
// 令牌不是领域概念（JWT、有效期策略属于认证基础设施），domain 不需要认识它。
type TokenIssuer interface {
	Issue(ctx context.Context, subject string, expire time.Duration) (token string, expireAt time.Time, err error)
}
