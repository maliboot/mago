package config

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	hertzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/cloudwego/hertz/pkg/route"
	rawjwt "github.com/golang-jwt/jwt/v4"
	hertzjwt "github.com/hertz-contrib/jwt"
)

// fakeUser 是最小 JWTUser 实现：身份键可配，Login 结果直接当载荷。
type fakeUser struct {
	identityKey string
	login       map[string]interface{}
	validate    func(identityVal string) bool
}

func (u *fakeUser) IdentityKey() string { return u.identityKey }

func (u *fakeUser) Login(username, password, ext string) map[string]interface{} {
	if username == "stone" && password == "pwd" {
		return u.login
	}

	return nil
}

func (u *fakeUser) Validate() func(identityVal string) bool { return u.validate }

func (u *fakeUser) ResponseWrapper(ctx context.Context, c *app.RequestContext, code int, message map[string]interface{}) {
	c.JSON(code, message)
}

func (u *fakeUser) UnauthorizedWrapper(ctx context.Context, c *app.RequestContext, code int, message string) {
	c.JSON(code, map[string]interface{}{"message": message})
}

// authenticatorUser 演示第二条登录入口：实现 JWTAuthenticator 后默认账密绑定被接管。
type authenticatorUser struct {
	*fakeUser
}

func (u *authenticatorUser) Authenticate(ctx context.Context, c *app.RequestContext) (map[string]interface{}, error) {
	if string(c.GetHeader("X-Login-Mode")) != "sms" {
		return nil, errors.New("不支持的登录方式")
	}

	return map[string]interface{}{
		"id":  "u-sms",
		"sub": "sms:13800000000",
		"ver": 2,
	}, nil
}

func newJWTConf(user JWTUser) *JWTConf {
	conf := &JWTConf{Secret: "unit-test-secret"}
	conf.ResetUserConfig(user)

	return conf
}

func newEngine() *route.Engine {
	return route.NewEngine(hertzconfig.NewOptions([]hertzconfig.Option{}))
}

func tokenClaims(t *testing.T, conf *JWTConf, token string) map[string]interface{} {
	t.Helper()

	parsed, err := conf.ParseTokenString(token)
	if err != nil {
		t.Fatalf("解析令牌失败: %v", err)
	}

	claims, ok := parsed.Claims.(rawjwt.MapClaims)
	if !ok {
		t.Fatalf("载荷类型不是 MapClaims: %T", parsed.Claims)
	}

	return claims
}

// 问题 1：载荷不能只剩身份键，sub/aid/ver 等业务字段要整体带上；TTL 按次指定不污染共享中间件。
func TestTokenGeneratorCarriesCustomClaimsWithScopedTTL(t *testing.T) {
	conf := newJWTConf(&fakeUser{identityKey: "id"})

	payload := map[string]interface{}{
		"id":  "u-1",
		"sub": "user:u-1",
		"aid": 42,
		"ver": 3,
	}

	token, expireAt, err := conf.TokenGeneratorWithTTL(payload, 2*time.Minute)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	claims := tokenClaims(t, conf, token)
	for key, want := range payload {
		if got := claims[key]; got != normalizeJSONNumber(want) {
			t.Fatalf("载荷 %q = %v，期望 %v", key, got, normalizeJSONNumber(want))
		}
	}

	// exp 由 hertz-jwt 写入，是 Unix 秒。
	exp, ok := claims["exp"].(float64)
	if !ok {
		t.Fatalf("exp 类型异常: %T", claims["exp"])
	}
	if delta := time.Until(time.Unix(int64(exp), 0)); delta > 2*time.Minute || delta < 90*time.Second {
		t.Fatalf("exp 与 2 分钟 TTL 不符: 剩余 %v", delta)
	}
	if delta := time.Until(expireAt); delta > 2*time.Minute || delta < 90*time.Second {
		t.Fatalf("返回的过期时间与 TTL 不符: 剩余 %v", delta)
	}

	// 共享中间件的有效期不能被按次 TTL 改掉。
	mw, err := conf.GetJwtMiddlewareCore()
	if err != nil {
		t.Fatalf("取中间件失败: %v", err)
	}
	if mw.Timeout != time.Hour {
		t.Fatalf("共享中间件 Timeout 被改成了 %v", mw.Timeout)
	}
}

// 保留键不能通过透传覆盖，否则会签出无法解析的令牌。
func TestTokenGeneratorDropsReservedClaims(t *testing.T) {
	conf := newJWTConf(&fakeUser{identityKey: "id"})

	token, _, err := conf.TokenGeneratorWithTTL(map[string]interface{}{
		"id":       "u-1",
		"exp":      1,
		"orig_iat": 2,
	}, time.Hour)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	claims := tokenClaims(t, conf, token)
	if exp, _ := claims["exp"].(float64); int64(exp) < time.Now().Unix() {
		t.Fatalf("业务传入的 exp 覆盖了框架写入的过期时间: %v", claims["exp"])
	}
	if origIat, _ := claims["orig_iat"].(float64); int64(origIat) < time.Now().Add(-time.Minute).Unix() {
		t.Fatalf("业务传入的 orig_iat 覆盖了框架写入的值: %v", claims["orig_iat"])
	}
}

// 缺身份键的载荷必须签发失败，而不是签出一个每个请求都会被拒的令牌。
func TestTokenGeneratorRejectsMissingIdentity(t *testing.T) {
	conf := newJWTConf(&fakeUser{identityKey: "id"})

	if _, _, err := conf.TokenGeneratorWithTTL(map[string]interface{}{"sub": "user:u-1"}, time.Hour); err == nil {
		t.Fatal("缺身份键应当报错")
	}
}

// 问题 1：中间件侧能读到 identity（数字型不 panic）与其它 claims；身份缺失不放行。
func TestMiddlewareIdentityAndClaims(t *testing.T) {
	conf := newJWTConf(&fakeUser{identityKey: "id"})

	token, _, err := conf.TokenGeneratorWithTTL(map[string]interface{}{
		"id":  42,
		"aid": "tenant-a",
	}, time.Hour)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	engine := newEngine()
	authed := engine.Group("/", conf.MiddlewareFunc())
	authed.GET("/me", func(ctx context.Context, c *app.RequestContext) {
		id, ok := conf.Identity(ctx, c)
		claims := conf.Claims(ctx, c)
		aid, _ := ClaimString(claims, "aid")
		c.JSON(http.StatusOK, map[string]interface{}{"id": id, "ok": ok, "aid": aid})
	})

	w := ut.PerformRequest(engine, http.MethodGet, "/me", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("数字型身份请求失败，code=%d body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"id":"42"`) || !strings.Contains(body, `"aid":"tenant-a"`) {
		t.Fatalf("身份或 claims 读取不对: %s", body)
	}

	// 令牌里没有身份键：既不能 panic，也不能放行。
	noIdentity, err := rawjwt.NewWithClaims(rawjwt.SigningMethodHS256, rawjwt.MapClaims{
		"sub": "user:u-1",
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(conf.Secret))
	if err != nil {
		t.Fatalf("构造令牌失败: %v", err)
	}

	w = ut.PerformRequest(engine, http.MethodGet, "/me", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + noIdentity})
	if w.Code == http.StatusOK {
		t.Fatalf("缺身份键的令牌被放行了: %s", w.Body.String())
	}
}

// Validate 返回 false 时请求必须被拒（动态作废令牌）。
func TestMiddlewareRejectsRevokedIdentity(t *testing.T) {
	conf := newJWTConf(&fakeUser{
		identityKey: "id",
		validate:    func(identityVal string) bool { return identityVal != "u-2" },
	})

	token, _, err := conf.TokenGeneratorWithTTL(map[string]interface{}{"id": "u-2"}, time.Hour)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	engine := newEngine()
	authed := engine.Group("/", conf.MiddlewareFunc())
	authed.GET("/me", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(http.StatusOK, map[string]interface{}{"ok": true})
	})

	if w := ut.PerformRequest(engine, http.MethodGet, "/me", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + token}); w.Code == http.StatusOK {
		t.Fatalf("已作废的身份被放行: %s", w.Body.String())
	}
}

// 问题 2：每种登录方式一条路由，各自校验凭证但共用载荷规则与响应。
func TestLoginHandlerWithCustomGrant(t *testing.T) {
	conf := newJWTConf(&fakeUser{identityKey: "id"})

	engine := newEngine()
	engine.POST("/sessions/email-code", conf.LoginHandlerWith(func(ctx context.Context, c *app.RequestContext) (interface{}, error) {
		var req struct {
			Email string `json:"email"`
			Code  string `json:"code"`
		}
		if err := c.BindAndValidate(&req); err != nil {
			return nil, errors.New("参数不完整")
		}
		if req.Code != "123456" {
			return nil, errors.New("验证码错误")
		}

		return map[string]interface{}{
			"id":  "u-9",
			"sub": req.Email,
			"ver": 1,
		}, nil
	}))

	body := []byte(`{"email":"stone@example.com","code":"123456"}`)
	w := ut.PerformRequest(engine, http.MethodPost, "/sessions/email-code",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"})
	if w.Code != http.StatusOK {
		t.Fatalf("邮箱验证码登录失败，code=%d body=%s", w.Code, w.Body.String())
	}

	var res struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil || res.Token == "" {
		t.Fatalf("登录响应里没有令牌: %s", w.Body.String())
	}

	claims := tokenClaims(t, conf, res.Token)
	if claims["sub"] != "stone@example.com" || claims["ver"] != float64(1) {
		t.Fatalf("自定义登录的载荷不对: %v", claims)
	}

	// 凭证错误走统一 401 包装。
	bad := []byte(`{"email":"stone@example.com","code":"000000"}`)
	w = ut.PerformRequest(engine, http.MethodPost, "/sessions/email-code",
		&ut.Body{Body: bytes.NewReader(bad), Len: len(bad)},
		ut.Header{Key: "Content-Type", Value: "application/json"})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("错误验证码应回 401，实际 code=%d body=%s", w.Code, w.Body.String())
	}

	// 载荷缺身份键时拒绝签发。
	engine.POST("/sessions/anonymous", conf.LoginHandlerWith(func(context.Context, *app.RequestContext) (interface{}, error) {
		return map[string]interface{}{"sub": "anonymous"}, nil
	}))
	w = ut.PerformRequest(engine, http.MethodPost, "/sessions/anonymous", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("缺身份键应回 401，实际 code=%d body=%s", w.Code, w.Body.String())
	}
}

// 问题 2：JWTUser 实现 JWTAuthenticator 后，同一个登录入口能处理多种凭证。
func TestLoginHandlerUsesJWTAuthenticator(t *testing.T) {
	conf := newJWTConf(&authenticatorUser{fakeUser: &fakeUser{identityKey: "id"}})
	engine := newEngine()
	engine.POST("/sessions", conf.LoginHandler())

	w := ut.PerformRequest(engine, http.MethodPost, "/sessions", nil,
		ut.Header{Key: "X-Login-Mode", Value: "sms"})
	if w.Code != http.StatusOK {
		t.Fatalf("短信登录失败，code=%d body=%s", w.Code, w.Body.String())
	}

	var res struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil || res.Token == "" {
		t.Fatalf("登录响应里没有令牌: %s", w.Body.String())
	}

	claims := tokenClaims(t, conf, res.Token)
	if claims["sub"] != "sms:13800000000" || claims["ver"] != float64(2) {
		t.Fatalf("JWTAuthenticator 的载荷不对: %v", claims)
	}

	// 不支持的登录方式仍然失败。
	if w := ut.PerformRequest(engine, http.MethodPost, "/sessions", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("未知登录方式应回 401，实际 code=%d", w.Code)
	}
}

// 默认账密登录维持原行为：username/password 绑定 + 载荷透传。
func TestDefaultLoginKeepsUsernamePassword(t *testing.T) {
	conf := newJWTConf(&fakeUser{
		identityKey: "id",
		login:       map[string]interface{}{"id": "u-1", "sub": "user:u-1"},
	})

	engine := newEngine()
	engine.POST("/sessions", conf.LoginHandler())

	body := []byte(`{"username":"stone","password":"pwd"}`)
	w := ut.PerformRequest(engine, http.MethodPost, "/sessions",
		&ut.Body{Body: bytes.NewReader(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"})
	if w.Code != http.StatusOK {
		t.Fatalf("账密登录失败，code=%d body=%s", w.Code, w.Body.String())
	}

	var res struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil || res.Token == "" {
		t.Fatalf("登录响应里没有令牌: %s", w.Body.String())
	}
	if claims := tokenClaims(t, conf, res.Token); claims["sub"] != "user:u-1" {
		t.Fatalf("账密登录的载荷不对: %v", claims)
	}
}

// ClaimsBuilder 覆盖载荷组装，给结构不规范（非 map）的登录结果留出口。
func TestSetClaimsBuilderOverridesPayload(t *testing.T) {
	conf := newJWTConf(&fakeUser{identityKey: "id"})
	conf.SetClaimsBuilder(func(data interface{}) hertzjwt.MapClaims {
		login, _ := data.(map[string]interface{})
		claims := hertzjwt.MapClaims{"ver": 9}
		for key, val := range login {
			claims[key] = val
		}

		return claims
	})

	token, _, err := conf.TokenGeneratorWithTTL(map[string]interface{}{"id": "u-1"}, time.Hour)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	if claims := tokenClaims(t, conf, token); claims["ver"] != float64(9) {
		t.Fatalf("ClaimsBuilder 未生效: %v", claims)
	}
}

// normalizeJSONNumber 按 JSON 往返后的类型比较，避免 int 与 float64 直接比较失败。
func normalizeJSONNumber(v interface{}) interface{} {
	raw, err := json.Marshal(v)
	if err != nil {
		return v
	}

	var out interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return v
	}

	return out
}

// responderUser 演示 JWTUser 自己决定登录成功响应字段（JWTLoginResponder）。
type responderUser struct {
	*fakeUser
	logins int
}

func (u *responderUser) LoginResponse(ctx context.Context, c *app.RequestContext, token string, expireAt time.Time, claims map[string]interface{}) {
	u.logins++
	c.JSON(http.StatusOK, map[string]interface{}{
		"accessToken": token,
		"identity":    claims["id"],
	})
}

// validatorUser 演示带 ctx 的动态校验（JWTValidator）。
type validatorUser struct {
	*fakeUser
	rejected string
}

func (u *validatorUser) ValidateIdentity(ctx context.Context, identityVal string) bool {
	return identityVal != u.rejected
}

// 自定义登录成功响应：JWTUser 实现 JWTLoginResponder 后字段自己定（默认只有 token/expire）。
func TestLoginResponseFromUserHook(t *testing.T) {
	user := &responderUser{fakeUser: &fakeUser{identityKey: "id"}}
	conf := newJWTConf(user)

	engine := newEngine()
	engine.POST("/sessions", conf.LoginHandlerWith(func(context.Context, *app.RequestContext) (interface{}, error) {
		return map[string]interface{}{"id": "u-1", "sub": "stone@example.com"}, nil
	}))

	w := ut.PerformRequest(engine, http.MethodPost, "/sessions", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("登录失败: code=%d body=%s", w.Code, w.Body.String())
	}
	if user.logins != 1 {
		t.Fatalf("自定义登录响应未被调用: %d", user.logins)
	}
	body := w.Body.String()
	if !strings.Contains(body, "accessToken") || !strings.Contains(body, `"identity":"u-1"`) {
		t.Fatalf("自定义响应字段不对: %s", body)
	}
	if strings.Contains(body, `"expire"`) {
		t.Fatalf("自定义响应不应再带默认字段: %s", body)
	}
}

// SetLoginResponder 优先级高于 JWTUser 的 LoginResponse，且能拿到完整载荷。
func TestSetLoginResponderOverridesUserHook(t *testing.T) {
	user := &responderUser{fakeUser: &fakeUser{identityKey: "id"}}
	conf := newJWTConf(user)

	var gotClaims map[string]interface{}
	conf.SetLoginResponder(func(ctx context.Context, c *app.RequestContext, token string, expireAt time.Time, claims map[string]interface{}) {
		gotClaims = claims
		c.JSON(http.StatusOK, map[string]interface{}{
			"token":  token,
			"userId": claims["id"],
			"email":  claims["sub"],
			"ver":    claims["ver"],
		})
	})

	engine := newEngine()
	engine.POST("/sessions", conf.LoginHandlerWith(func(context.Context, *app.RequestContext) (interface{}, error) {
		return map[string]interface{}{"id": "u-2", "sub": "stone@example.com", "ver": 7}, nil
	}))

	w := ut.PerformRequest(engine, http.MethodPost, "/sessions", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("登录失败: code=%d body=%s", w.Code, w.Body.String())
	}
	if user.logins != 0 {
		t.Fatal("配置级 Responder 应覆盖 JWTUser 的 LoginResponse")
	}
	if gotClaims["sub"] != "stone@example.com" {
		t.Fatalf("Responder 拿到的载荷不对: %v", gotClaims)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"userId":"u-2"`) || !strings.Contains(body, `"email":"stone@example.com"`) {
		t.Fatalf("自定义响应字段不对: %s", body)
	}
}

// JWTValidator 带 ctx 校验：被拒绝的身份必须拦下。
func TestValidateIdentityUsesContext(t *testing.T) {
	user := &validatorUser{fakeUser: &fakeUser{identityKey: "id"}, rejected: "u-3"}
	conf := newJWTConf(user)

	token, _, err := conf.TokenGeneratorWithTTL(map[string]interface{}{"id": "u-3"}, time.Hour)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}

	engine := newEngine()
	authed := engine.Group("/", conf.MiddlewareFunc())
	authed.GET("/me", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(http.StatusOK, map[string]interface{}{"ok": true})
	})

	if w := ut.PerformRequest(engine, http.MethodGet, "/me", nil,
		ut.Header{Key: "Authorization", Value: "Bearer " + token}); w.Code == http.StatusOK {
		t.Fatalf("ValidateIdentity 拒绝的身份被放行: %s", w.Body.String())
	}
}

// LoginHandlerWithTTL 用指定有效期签发，且不改共享中间件。
func TestLoginHandlerWithTTL(t *testing.T) {
	conf := newJWTConf(&fakeUser{identityKey: "id"})

	engine := newEngine()
	engine.POST("/sessions", conf.LoginHandlerWithTTL(func(context.Context, *app.RequestContext) (interface{}, error) {
		return map[string]interface{}{"id": "u-4"}, nil
	}, 3*time.Hour))

	w := ut.PerformRequest(engine, http.MethodPost, "/sessions", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("登录失败: code=%d body=%s", w.Code, w.Body.String())
	}

	var res struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil || res.Token == "" {
		t.Fatalf("登录响应里没有令牌: %s", w.Body.String())
	}

	exp, ok := ClaimInt64(tokenClaims(t, conf, res.Token), "exp")
	if !ok {
		t.Fatal("令牌缺少 exp")
	}
	if delta := time.Until(time.Unix(exp, 0)); delta > 3*time.Hour || delta < 3*time.Hour-time.Minute {
		t.Fatalf("TTL 未生效: 剩余 %v", delta)
	}

	mw, err := conf.GetJwtMiddlewareCore()
	if err != nil {
		t.Fatalf("取中间件失败: %v", err)
	}
	if mw.Timeout != time.Hour {
		t.Fatalf("共享中间件 Timeout 被改成了 %v", mw.Timeout)
	}
}

// LoginResult 让同一个登录入口按请求决定有效期（如「延长登录」），且不改共享中间件。
func TestLoginResultPerRequestTTL(t *testing.T) {
	conf := newJWTConf(&fakeUser{identityKey: "id"})

	engine := newEngine()
	engine.POST("/sessions", conf.LoginHandlerWith(func(ctx context.Context, c *app.RequestContext) (interface{}, error) {
		ttl := time.Hour
		if string(c.GetHeader("X-Extend")) == "true" {
			ttl = 48 * time.Hour
		}

		return LoginResult{Claims: map[string]interface{}{"id": "u-5"}, TTL: ttl}, nil
	}))

	for _, extend := range []bool{false, true} {
		headers := []ut.Header{}
		if extend {
			headers = append(headers, ut.Header{Key: "X-Extend", Value: "true"})
		}

		w := ut.PerformRequest(engine, http.MethodPost, "/sessions", nil, headers...)
		if w.Code != http.StatusOK {
			t.Fatalf("登录失败(extend=%v): code=%d body=%s", extend, w.Code, w.Body.String())
		}

		var res struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil || res.Token == "" {
			t.Fatalf("登录响应里没有令牌: %s", w.Body.String())
		}

		exp, ok := ClaimInt64(tokenClaims(t, conf, res.Token), "exp")
		if !ok {
			t.Fatal("令牌缺少 exp")
		}
		want := time.Hour
		if extend {
			want = 48 * time.Hour
		}
		if delta := time.Until(time.Unix(exp, 0)); delta > want || delta < want-time.Minute {
			t.Fatalf("extend=%v 的 TTL 不对: 剩余 %v，期望约 %v", extend, delta, want)
		}
	}

	mw, err := conf.GetJwtMiddlewareCore()
	if err != nil {
		t.Fatalf("取中间件失败: %v", err)
	}
	if mw.Timeout != time.Hour {
		t.Fatalf("共享中间件 Timeout 被改成了 %v", mw.Timeout)
	}
}

// 自定义登录入口的凭证错误仍走统一 401 包装（不因为改成先校验而变形）。
func TestLoginHandlerWithKeepsUnauthorizedShape(t *testing.T) {
	conf := newJWTConf(&fakeUser{identityKey: "id"})

	engine := newEngine()
	engine.POST("/sessions", conf.LoginHandlerWith(func(context.Context, *app.RequestContext) (interface{}, error) {
		return nil, errors.New("验证码错误")
	}))

	w := ut.PerformRequest(engine, http.MethodPost, "/sessions", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("应回 401，实际 code=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "验证码错误") {
		t.Fatalf("401 文案丢失: %s", w.Body.String())
	}
}
