package util

import (
	"context"

	"golang.org/x/crypto/bcrypt"
)

// Password 是 domain.PasswordHasher 的 bcrypt 实现。
type Password struct{}

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
