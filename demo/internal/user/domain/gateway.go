package domain

import "context"

// UserGateway 是用户聚合的出站端口，由 infrastructure 实现，只服务于写路径。
// 只有聚合自身的持久化契约放 domain；散列、令牌签发这类技术能力端口由使用方（application）定义。
type UserGateway interface {
	GetByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
}
