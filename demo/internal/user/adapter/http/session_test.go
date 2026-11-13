package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	hertzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/route"
	"github.com/maliboot/mago"
	"github.com/maliboot/mago/config"

	"tests/internal/user/application"
	"tests/internal/user/client/dto"
	"tests/internal/user/client/enum"
	"tests/internal/user/client/vo"
)

// stubService 是 application.UserService 的测试替身：用内存验证码跑完整的「发码 → 登录 → 带令牌访问」链路。
type stubService struct {
	codes   map[string]string
	active  bool
	meCalls int
}

func newStubService() *stubService {
	return &stubService{codes: make(map[string]string), active: true}
}

func (s *stubService) Login(context.Context, *dto.UserAuthRequest) *mago.VO[vo.UserTokenVO] {
	return mago.NewVO[vo.UserTokenVO](nil, vo.NewUserTokenVO("password-token", "2026-11-10 12:00:00"))
}

func (s *stubService) LoginByPassword(_ context.Context, req *dto.UserAuthRequest) *mago.VO[vo.LoginSubjectVO] {
	if req.Password != "secret" {
		return mago.NewVO[vo.LoginSubjectVO](mago.NewError(mago.ErrAuthLoginFailed).WithMsg("邮箱或密码错误"), nil)
	}

	return mago.NewVO[vo.LoginSubjectVO](nil, vo.NewLoginSubjectVO(1, req.Email, 1))
}

func (s *stubService) SendVerifyCode(_ context.Context, req *dto.VerifyCodeRequest) *mago.VO[mago.EmptyVO] {
	if req.Email == "" {
		return mago.NewVO[mago.EmptyVO](mago.NewError(mago.ErrInvalidParams).WithMsg("邮箱不能为空"), nil)
	}
	// 假发信：验证码直接留在内存里，用例随后用它登录。
	s.codes[req.Email] = "123456"

	return mago.NewVO[mago.EmptyVO](nil, nil)
}

func (s *stubService) LoginByEmailCode(_ context.Context, req *dto.EmailCodeLoginRequest) *mago.VO[vo.LoginSubjectVO] {
	if code, ok := s.codes[req.Email]; !ok || code != req.Code {
		return mago.NewVO[vo.LoginSubjectVO](mago.NewError(mago.ErrAuthLoginFailed).WithMsg("验证码错误或已过期"), nil)
	}
	delete(s.codes, req.Email)

	return mago.NewVO[vo.LoginSubjectVO](nil, vo.NewLoginSubjectVO(1, req.Email, 1))
}

func (s *stubService) Register(context.Context, *dto.UserAuthRequest) *mago.VO[mago.EmptyVO] {
	return mago.NewVO[mago.EmptyVO](nil, nil)
}

func (s *stubService) Me(_ context.Context, userID uint) *mago.VO[vo.UserVO] {
	s.meCalls++
	if userID != 1 {
		return mago.NewVO[vo.UserVO](mago.NewError(mago.ErrUriNotFound).WithMsg("用户不存在"), nil)
	}

	status := enum.UserStatusDisabled
	if s.active {
		status = enum.UserStatusActive
	}

	return mago.NewVO[vo.UserVO](nil, vo.NewUserVO("stone@example.com", status))
}

// newSessionEngine 装配与组合根一致的登录链路：JWTConf（含业务扩展）+ 控制器 + 路由分组。
func newSessionEngine(service application.UserService) (*route.Engine, *config.JWTConf) {
	jwtConf := &config.JWTConf{Secret: "session-test-secret"}
	jwtAuth := NewJWTAuth(service)
	jwtConf.ResetUserConfig(jwtAuth)

	controller := NewUserController(service, jwtConf)

	engine := route.NewEngine(hertzconfig.NewOptions([]hertzconfig.Option{}))
	v1 := engine.Group("/api/v1/user")
	authed := v1.Group("", jwtConf.MiddlewareFunc())
	controller.Router(v1, authed)

	return engine, jwtConf
}

func postJSON(engine *route.Engine, path string, body []byte) *ut.ResponseRecorder {
	return ut.PerformRequest(engine, http.MethodPost, path,
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"})
}

// TestEmailCodeLoginFlow 覆盖验收链路：发码 → 验证码登录（自定义响应字段）→ 带令牌访问受保护接口。
func TestEmailCodeLoginFlow(t *testing.T) {
	service := newStubService()
	engine, _ := newSessionEngine(service)

	if w := postJSON(engine, "/api/v1/user/users/verify-code", []byte(`{"email":"stone@example.com"}`)); w.Code != http.StatusOK {
		t.Fatalf("发码失败：code=%d body=%s", w.Code, w.Body.String())
	}

	w := postJSON(engine, "/api/v1/user/sessions/emailCode", []byte(`{"email":"stone@example.com","code":"123456"}`))
	if w.Code != http.StatusOK {
		t.Fatalf("验证码登录失败：code=%d body=%s", w.Code, w.Body.String())
	}

	var login struct {
		Code int `json:"code"`
		Data struct {
			Token        string `json:"token"`
			Expire       string `json:"expire"`
			UserID       string `json:"userId"`
			Email        string `json:"email"`
			TokenVersion int    `json:"tokenVersion"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &login); err != nil {
		t.Fatalf("登录响应解析失败：%v body=%s", err, w.Body.String())
	}
	if login.Data.Token == "" {
		t.Fatalf("登录响应缺令牌：%s", w.Body.String())
	}
	// 自定义成功响应字段：除 token/expire 外带回登录主体。
	if login.Data.UserID != "1" || login.Data.Email != "stone@example.com" || login.Data.TokenVersion != 1 {
		t.Fatalf("自定义响应字段不对：%s", w.Body.String())
	}

	// 带令牌访问受保护接口：中间件取 claims[id] → ValidateIdentity → 放行。
	w = ut.PerformRequest(engine, http.MethodGet, "/api/v1/user/users/1", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + login.Data.Token})
	if w.Code != http.StatusOK {
		t.Fatalf("带令牌访问失败：code=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "stone@example.com") {
		t.Fatalf("用户信息不对：%s", w.Body.String())
	}

	// 停用后旧令牌立刻失效（ValidateIdentity 查状态）。
	service.active = false
	w = ut.PerformRequest(engine, http.MethodGet, "/api/v1/user/users/1", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + login.Data.Token})
	if w.Code == http.StatusOK {
		t.Fatalf("停用用户的旧令牌被放行：%s", w.Body.String())
	}
}

// TestEmailCodeLoginRejectsWrongCode 错误验证码走统一 401 响应形状。
func TestEmailCodeLoginRejectsWrongCode(t *testing.T) {
	service := newStubService()
	engine, _ := newSessionEngine(service)

	postJSON(engine, "/api/v1/user/users/verify-code", []byte(`{"email":"stone@example.com"}`))

	w := postJSON(engine, "/api/v1/user/sessions/emailCode", []byte(`{"email":"stone@example.com","code":"000000"}`))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("错误验证码应回 401：code=%d body=%s", w.Code, w.Body.String())
	}

	var res struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("401 响应解析失败：%v body=%s", err, w.Body.String())
	}
	// UnauthorizedWrapper 把 401 统一成 mago 错误码（401 段），并保留验证码失败文案。
	if res.Code/1000 != http.StatusUnauthorized || !strings.Contains(res.Msg, "验证码") {
		t.Fatalf("401 响应形状不对：%s", w.Body.String())
	}
}

// TestEmailCodeLoginOneTime 验证码只能换一次令牌。
func TestEmailCodeLoginOneTime(t *testing.T) {
	service := newStubService()
	engine, _ := newSessionEngine(service)

	postJSON(engine, "/api/v1/user/users/verify-code", []byte(`{"email":"stone@example.com"}`))

	first := postJSON(engine, "/api/v1/user/sessions/emailCode", []byte(`{"email":"stone@example.com","code":"123456"}`))
	if first.Code != http.StatusOK {
		t.Fatalf("首次登录失败：code=%d body=%s", first.Code, first.Body.String())
	}

	second := postJSON(engine, "/api/v1/user/sessions/emailCode", []byte(`{"email":"stone@example.com","code":"123456"}`))
	if second.Code == http.StatusOK {
		t.Fatalf("同一个验证码被重复使用：%s", second.Body.String())
	}
}

// TestEmailCodeLoginRequiresTokenForAuthedRoute 未带令牌访问受保护接口必须 401。
func TestEmailCodeLoginRequiresTokenForAuthedRoute(t *testing.T) {
	engine, _ := newSessionEngine(newStubService())

	w := ut.PerformRequest(engine, http.MethodGet, "/api/v1/user/users/1", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("未带令牌应回 401：code=%d body=%s", w.Code, w.Body.String())
	}
}

// loginResponse 是两种登录共用的成功响应形状（自定义字段由 JWTAuth.LoginResponse 给出）。
type loginResponse struct {
	Code int `json:"code"`
	Data struct {
		Token        string `json:"token"`
		Expire       string `json:"expire"`
		UserID       string `json:"userId"`
		Email        string `json:"email"`
		TokenVersion int    `json:"tokenVersion"`
	} `json:"data"`
	Msg string `json:"msg"`
}

func decodeLogin(t *testing.T, w *ut.ResponseRecorder) loginResponse {
	t.Helper()

	var res loginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("登录响应解析失败：%v body=%s", err, w.Body.String())
	}

	return res
}

// TestPasswordLoginFlow 密码登录也走中间件入口：自定义响应字段 + 令牌可访问受保护接口。
func TestPasswordLoginFlow(t *testing.T) {
	service := newStubService()
	engine, _ := newSessionEngine(service)

	w := postJSON(engine, "/api/v1/user/sessions", []byte(`{"email":"stone@example.com","password":"secret"}`))
	if w.Code != http.StatusOK {
		t.Fatalf("密码登录失败：code=%d body=%s", w.Code, w.Body.String())
	}

	login := decodeLogin(t, w)
	if login.Data.Token == "" || login.Data.UserID != "1" || login.Data.Email != "stone@example.com" || login.Data.TokenVersion != 1 {
		t.Fatalf("密码登录响应字段不对：%s", w.Body.String())
	}

	w = ut.PerformRequest(engine, http.MethodGet, "/api/v1/user/users/1", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + login.Data.Token})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "stone@example.com") {
		t.Fatalf("密码登录的令牌访问受保护接口失败：code=%d body=%s", w.Code, w.Body.String())
	}
}

// TestPasswordLoginExtendExpire X-Iee: true 时按请求延长有效期（LoginResult.TTL），且不污染默认入口。
func TestPasswordLoginExtendExpire(t *testing.T) {
	engine, _ := newSessionEngine(newStubService())
	body := []byte(`{"email":"stone@example.com","password":"secret"}`)

	normal := decodeLogin(t, postJSON(engine, "/api/v1/user/sessions", body))
	extended := decodeLogin(t, ut.PerformRequest(engine, http.MethodPost, "/api/v1/user/sessions",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
		ut.Header{Key: "X-Iee", Value: "true"}))

	for name, res := range map[string]loginResponse{"默认": normal, "延长": extended} {
		expireAt, err := time.ParseInLocation(time.DateTime, res.Data.Expire, time.Local)
		if err != nil {
			t.Fatalf("%s登录的 expire 解析失败：%v body=%+v", name, err, res.Data)
		}
		want := application.DefaultTokenTTL
		if name == "延长" {
			want = application.ExtendTokenTTL
		}
		if delta := time.Until(expireAt); delta > want || delta < want-time.Minute {
			t.Fatalf("%s登录有效期不对：剩余 %v，期望约 %v", name, delta, want)
		}
	}
}

// TestPasswordLoginRejectsWrongPassword 密码错误走统一 401 形状，并保留业务文案。
func TestPasswordLoginRejectsWrongPassword(t *testing.T) {
	engine, _ := newSessionEngine(newStubService())

	w := postJSON(engine, "/api/v1/user/sessions", []byte(`{"email":"stone@example.com","password":"wrong"}`))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("密码错误应回 401：code=%d body=%s", w.Code, w.Body.String())
	}

	var res struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("401 响应解析失败：%v body=%s", err, w.Body.String())
	}
	if res.Code/1000 != http.StatusUnauthorized || !strings.Contains(res.Msg, "密码") {
		t.Fatalf("401 响应形状不对：%s", w.Body.String())
	}
}
