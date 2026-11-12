package repository

import (
	"context"
	"errors"

	"github.com/maliboot/mago"
	"github.com/maliboot/mago/config"
	"gorm.io/gorm"

	"tests/internal/user/client/vo"
	"tests/internal/user/domain"
	"tests/internal/user/infrastructure/convertor"
	"tests/internal/user/infrastructure/do/db"
)

// User 是用户 DO 唯一的 repository：写路径实现 domain.UserGateway，
// 读路径实现 application.UserQuery（轻量 CQRS），转换交给 convertor。
// 所有方法都接收 ctx，并透传到 gorm（WithContext），保证超时与取消能一路传下去。
type User struct {
	mago.AbstractRepository[db.User]
}

// 非单例：按配置装配 repository，由组合根创建一次，每上下文共用同一实例。
func NewUser(c *config.Conf) *User {
	ins := &User{}
	ins.Conf = c
	return ins
}

func (r *User) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var do db.User

	tx := r.DB(do).WithContext(ctx).Select("*").Where("email = ?", domain.NormalizeEmail(email)).First(&do)
	if err := tx.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, mago.NewError(mago.ErrUriNotFound).WithMsg("用户不存在")
		}

		return nil, err
	}

	return convertor.UserToEntity(&do), nil
}

// ByID 是读路径：只取展示需要的列并组装 VO，不加载领域聚合。
func (r *User) ByID(ctx context.Context, id uint) (*vo.UserVO, error) {
	if id == 0 {
		return nil, mago.NewError(mago.ErrUriNotFound).WithMsg("用户不存在")
	}

	var do db.User

	tx := r.DB(do).WithContext(ctx).Select("*").Where("id = ?", id).First(&do)
	if err := tx.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, mago.NewError(mago.ErrUriNotFound).WithMsg("用户不存在")
		}

		return nil, err
	}

	return convertor.UserToVO(&do), nil
}

func (r *User) Create(ctx context.Context, user *domain.User) error {
	do := convertor.UserToDO(user)

	return r.DB(*do).WithContext(ctx).Create(do).Error
}

func (r *User) Update(ctx context.Context, user *domain.User) error {
	do := convertor.UserToDO(user)

	return r.DB(*do).WithContext(ctx).Where("id = ?", user.ID).Updates(do).Error
}
