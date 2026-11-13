package application

import (
	"context"
	"regexp"
	"testing"
	"time"

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

// fakeCodeStore 用内存 map 存码，避免用例依赖基础设施实现。
type fakeCodeStore struct {
	codes map[string]string
	err   error
}

func (f *fakeCodeStore) Save(_ context.Context, email, code string, _ time.Duration) error {
	if f.err != nil {
		return f.err
	}
	if f.codes == nil {
		f.codes = make(map[string]string)
	}
	f.codes[email] = code

	return nil
}

func (f *fakeCodeStore) Consume(_ context.Context, email, code string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	if f.codes[email] == "" || f.codes[email] != code {
		return false, nil
	}
	delete(f.codes, email)

	return true, nil
}

// newServiceForTest 用替身构造只测验证码用例的 service。
func newServiceForTest(sender CodeSender, store VerifyCodeStore) *userService {
	return &userService{codes: sender, verifyCodes: store}
}

func TestSendVerifyCode(t *testing.T) {
	sender := &fakeCodeSender{}
	store := &fakeCodeStore{}
	service := newServiceForTest(sender, store)

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
	if store.codes[sender.email] != sender.code {
		t.Fatalf("验证码未落存储：store=%v, 发送=%q", store.codes, sender.code)
	}
}

func TestSendVerifyCodeRejectsEmptyEmail(t *testing.T) {
	sender := &fakeCodeSender{}
	service := newServiceForTest(sender, &fakeCodeStore{})

	res := service.SendVerifyCode(context.Background(), &dto.VerifyCodeRequest{Email: "   "})
	if res.Error() == nil {
		t.Fatal("空邮箱应返回错误")
	}
	if sender.code != "" {
		t.Fatal("空邮箱不应触发发送")
	}
}
