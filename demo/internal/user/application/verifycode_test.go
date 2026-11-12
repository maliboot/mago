package application

import (
	"context"
	"regexp"
	"testing"

	"tests/internal/user/client/dto"
)

// fakeCodeSender 记录最近一次发送的验证码，避免用例依赖真实 SMTP。
type fakeCodeSender struct {
	email string
	code  string
	err   error
}

func (f *fakeCodeSender) SendCode(_ context.Context, email, code string) error {
	f.email = email
	f.code = code

	return f.err
}

// newServiceForTest 用替身构造只测验证码用例的 service。
func newServiceForTest(sender CodeSender) *userService {
	return &userService{codes: sender}
}

func TestSendVerifyCode(t *testing.T) {
	sender := &fakeCodeSender{}
	service := newServiceForTest(sender)

	res := service.SendVerifyCode(context.Background(), &dto.VerifyCodeRequest{Email: "  User@Example.COM "})
	if err := res.Error(); err != nil {
		t.Fatalf("SendVerifyCode: %v", err)
	}

	if sender.email != "user@example.com" {
		t.Fatalf("收件人 = %q, 期望规范化后的 user@example.com", sender.email)
	}
	if !regexp.MustCompile(`^\d{6}$`).MatchString(sender.code) {
		t.Fatalf("验证码 = %q, 期望 6 位数字", sender.code)
	}
}

func TestSendVerifyCodeRejectsEmptyEmail(t *testing.T) {
	sender := &fakeCodeSender{}
	service := newServiceForTest(sender)

	res := service.SendVerifyCode(context.Background(), &dto.VerifyCodeRequest{Email: "   "})
	if res.Error() == nil {
		t.Fatal("空邮箱应返回错误")
	}
	if sender.code != "" {
		t.Fatal("空邮箱不应触发发送")
	}
}
