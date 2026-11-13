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

// VerifyCodeStore 保存与核销邮箱验证码；由 infrastructure 的内存或 Redis 实现提供。
// 与 CodeSender 分开：发得出去不等于存得住，两者可以各自替换（§7.4）。
// ⚠️ demo 默认注入的是内存单副本实现（repository.VerifyCode），仅供本地联调；
// 生产必须换 Redis / 数据库实现（多副本共享、重启不丢、可加错误次数上限），
// 只改组合根 conf/provider.go 的 provideVerifyCodeStore 一行，本端口与用例不变。
type VerifyCodeStore interface {
	Save(ctx context.Context, email, code string, ttl time.Duration) error
	Consume(ctx context.Context, email, code string) (bool, error)
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
