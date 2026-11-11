package api

import (
	"context"

	"tests/internal/user/client/vo"
)

// UserService 是 user 上下文对其他上下文暴露的用例契约。
// 其他上下文只依赖本包与 client/dto、client/vo，不依赖 user 的 application、domain、infrastructure。
type UserService interface {
	// FindByID 按 ID 返回用户视图，找不到时返回 mago.ErrUriNotFound。
	FindByID(ctx context.Context, id uint) (*vo.UserVO, error)
}
