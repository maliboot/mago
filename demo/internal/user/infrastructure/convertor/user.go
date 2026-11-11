package convertor

import (
	"time"

	"tests/internal/user/client/enum"
	"tests/internal/user/client/vo"
	"tests/internal/user/domain"
	"tests/internal/user/infrastructure/do/db"
	"tests/internal/user/infrastructure/util"
)

// UserToEntity 把用户 DO 还原成领域聚合。
func UserToEntity(do *db.User) *domain.User {
	var lastLoginAt *time.Time
	if do.LastLoginAt != nil {
		lastLoginAt = do.LastLoginAt
	}

	return domain.RestoreUser(
		uint(do.ID),
		do.Email,
		do.PasswordHash,
		do.Name,
		util.Int8ToBool(do.IsActive),
		lastLoginAt,
	)
}

// UserToDO 把领域聚合展开成用户 DO。
func UserToDO(u *domain.User) *db.User {
	do := &db.User{
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Name:         u.Name,
		IsActive:     util.BoolToInt8(u.IsActive),
	}
	do.ID = int64(u.ID)
	if u.LastLoginAt != nil {
		do.LastLoginAt = u.LastLoginAt
	}

	return do
}

// UserToVO 把用户 DO 直接组装成对外视图，供读路径使用，不经过领域聚合。
func UserToVO(do *db.User) *vo.UserVO {
	return vo.NewUserVO(do.Email, enum.FromActive(util.Int8ToBool(do.IsActive)))
}
