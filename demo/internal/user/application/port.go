package application

import (
	"context"
	"time"

	"github.com/maliboot/mago/dispatcher"
)

// PasswordHasher 是密码散列能力端口，由 infrastructure 实现。
// 使用方是 application（用例），所以按「接口由使用方定义」放在这里，而不是 domain。
type PasswordHasher interface {
	Hash(ctx context.Context, raw string) (string, error)
	Verify(ctx context.Context, raw, hash string) bool
}

// CodeSender 把验证码发送到用户邮箱；由 infrastructure 的邮件适配器实现（SMTP、短信、钉钉可各自实现）。
type CodeSender interface {
	SendCode(ctx context.Context, email, code string) error
}

// TokenIssuer 是访问令牌签发能力端口，由 infrastructure 实现。
// 令牌不是领域概念（JWT、有效期策略属于认证基础设施），domain 不需要认识它。
type TokenIssuer interface {
	Issue(ctx context.Context, subject string, expire time.Duration) (token string, expireAt time.Time, err error)
}

// TaskQueue 是异步任务发布端口（技术能力端口，消费方定义）：
// 实现走 infrastructure 的 mago queuex（优先 redis 实现），见 README §12.3。
type TaskQueue interface {
	Enqueue(ctx context.Context, task *dispatcher.Task) error
}
