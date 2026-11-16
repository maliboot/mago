package util

import (
	"context"

	"golang.org/x/crypto/bcrypt"
)

// Password 是 application.PasswordHasher 的 bcrypt 实现。
// 无 DO 的纯工具类：无状态纯计算，不操作任何数据对象，按 README §18.6 放 util/，不占 repository/。
type Password struct{}

// 非单例：无状态纯实现，随手构造即可，做成单例没有收益。
func NewPassword() *Password {
	return &Password{}
}

// Hash 生成密码散列，成本因子使用 bcrypt 默认值。纯计算，不消费 ctx。
func (p *Password) Hash(_ context.Context, raw string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	return string(hashed), err
}

// Verify 校验明文密码与散列是否匹配。纯计算，不消费 ctx。
func (p *Password) Verify(_ context.Context, raw, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(raw)) == nil
}
