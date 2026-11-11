package domain_test

import (
	"errors"
	"testing"

	"github.com/maliboot/mago"

	"tests/internal/user/domain"
)

// TestNewUserReturnsGlobalErrorCode 确认领域校验用的是框架全局错误码，而不是自定义错误。
func TestNewUserReturnsGlobalErrorCode(t *testing.T) {
	_, err := domain.NewUser("  ", "hash", "n")
	if err == nil {
		t.Fatal("邮箱为空时应返回错误")
	}

	var e mago.Error
	if !errors.As(err, &e) {
		t.Fatalf("期望 mago.Error，实际为 %T", err)
	}
	if e.Code() != mago.ErrInvalidParams {
		t.Fatalf("期望 ErrInvalidParams，实际 %v", e.Code())
	}

	_, err = domain.NewUser("a@b.com", "", "n")
	if !errors.As(err, &e) || e.Code() != mago.ErrInvalidParams {
		t.Fatalf("密码为空应返回 ErrInvalidParams，实际 %v", err)
	}
}

// TestNewUserNormalizesEmail 确认邮箱规范化生效。
func TestNewUserNormalizesEmail(t *testing.T) {
	u, err := domain.NewUser("  Foo@Bar.COM ", "hash", " 张三 ")
	if err != nil {
		t.Fatalf("创建用户失败：%v", err)
	}
	if u.Email != "foo@bar.com" {
		t.Fatalf("邮箱未规范化：%s", u.Email)
	}
	if u.Name != "张三" {
		t.Fatalf("名称未去空白：%q", u.Name)
	}
	if !u.IsActive {
		t.Fatal("新用户默认应为启用状态")
	}
}

// TestCanLoginRejectsDisabledUser 确认停用账号无法登录，且错误码取自框架。
func TestCanLoginRejectsDisabledUser(t *testing.T) {
	u := domain.RestoreUser(1, "a@b.com", "hash", "n", false, nil)

	var e mago.Error
	if err := u.CanLogin(); !errors.As(err, &e) || e.Code() != mago.ErrAuthFailed {
		t.Fatalf("期望 ErrAuthFailed，实际 %v", err)
	}
}
