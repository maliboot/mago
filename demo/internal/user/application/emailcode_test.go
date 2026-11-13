package application

import (
	"context"
	"testing"

	"tests/internal/user/client/dto"
	"tests/internal/user/domain"
)

// activeUser 返回一个可用于登录的存量用户。
func activeUser() *domain.User {
	return domain.RestoreUser(7, "stone@example.com", "hash", "Stone", true, nil)
}

func TestLoginByEmailCode(t *testing.T) {
	store := &fakeCodeStore{codes: map[string]string{"stone@example.com": "123456"}}
	service := &userService{users: &fakeUserGateway{existing: activeUser()}, verifyCodes: store}

	res := service.LoginByEmailCode(context.Background(), &dto.EmailCodeLoginRequest{
		Email: " Stone@Example.com ",
		Code:  "123456",
	})
	if err := res.Error(); err != nil {
		t.Fatalf("LoginByEmailCode: %v", err)
	}
	if res.Data.UserID != 7 || res.Data.Email != "stone@example.com" {
		t.Fatalf("登录主体不对: %+v", res.Data)
	}
	if res.Data.TokenVersion != 1 {
		t.Fatalf("令牌版本 = %d，期望 1", res.Data.TokenVersion)
	}
	if _, left := store.codes["stone@example.com"]; left {
		t.Fatal("验证码登录成功后必须核销，不能重复使用")
	}
}

func TestLoginByEmailCodeRejectsWrongCode(t *testing.T) {
	store := &fakeCodeStore{codes: map[string]string{"stone@example.com": "123456"}}
	service := &userService{users: &fakeUserGateway{existing: activeUser()}, verifyCodes: store}

	res := service.LoginByEmailCode(context.Background(), &dto.EmailCodeLoginRequest{
		Email: "stone@example.com",
		Code:  "000000",
	})
	if res.Error() == nil {
		t.Fatal("错误验证码必须拒绝")
	}
	if _, left := store.codes["stone@example.com"]; !left {
		t.Fatal("填错验证码不应核销，用户还能用正确验证码再试")
	}
}

func TestLoginByEmailCodeRejectsUnknownEmail(t *testing.T) {
	store := &fakeCodeStore{codes: map[string]string{"ghost@example.com": "123456"}}
	service := &userService{users: &fakeUserGateway{}, verifyCodes: store}

	res := service.LoginByEmailCode(context.Background(), &dto.EmailCodeLoginRequest{
		Email: "ghost@example.com",
		Code:  "123456",
	})
	if res.Error() == nil {
		t.Fatal("未注册邮箱不能靠验证码登录")
	}
}

func TestLoginByEmailCodeRejectsDisabledUser(t *testing.T) {
	disabled := domain.RestoreUser(8, "off@example.com", "hash", "Off", false, nil)
	store := &fakeCodeStore{codes: map[string]string{"off@example.com": "123456"}}
	service := &userService{users: &fakeUserGateway{existing: disabled}, verifyCodes: store}

	res := service.LoginByEmailCode(context.Background(), &dto.EmailCodeLoginRequest{
		Email: "off@example.com",
		Code:  "123456",
	})
	if res.Error() == nil {
		t.Fatal("停用账号不能登录")
	}
}

func TestLoginByEmailCodeRejectsEmptyFields(t *testing.T) {
	service := &userService{users: &fakeUserGateway{}, verifyCodes: &fakeCodeStore{}}

	if res := service.LoginByEmailCode(context.Background(), &dto.EmailCodeLoginRequest{Email: "a@b.com"}); res.Error() == nil {
		t.Fatal("空验证码应返回参数错误")
	}
	if res := service.LoginByEmailCode(context.Background(), &dto.EmailCodeLoginRequest{Code: "123456"}); res.Error() == nil {
		t.Fatal("空邮箱应返回参数错误")
	}
}
