package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	rawjwt "github.com/golang-jwt/jwt/v4"
	"github.com/hertz-contrib/jwt"
)

const (
	// claimExp / claimOrigIat 由 hertz-jwt 在签发时写入，业务载荷里的同名键会被丢弃。
	claimExp     = "exp"
	claimOrigIat = "orig_iat"

	// defaultIdentityKey 未指定 IdentityKey 时使用的身份键。
	defaultIdentityKey = "id"
)

type JWTUser interface {
	// IdentityKey 唯一标识，如id。必须
	IdentityKey() string
	// Login 默认登录方式：用户名 + 密码。当需要自定义[jwt.LoginHandler]登录逻辑，则可以重写该函数。否则是是可选的。
	// 邮箱+验证码、手机+短信验证码、第三方快捷登录不适用这条路径：
	// 实现 JWTAuthenticator 由它按请求分派，或给每种登录方式用 JWTConf.LoginHandlerWith 单独挂入口。
	// 该函数接收三个参数：username（用户名），password（密码），ext（扩展信息）。
	// 它返回一个映射，其中包含登录结果的相关信息。
	// 参数:
	//   - username: 用户名字符串，用于标识用户。
	//   - password: 密码字符串，用于验证用户身份。
	//   - ext: 扩展信息字符串，可用于额外的验证或跟踪信息。
	// 返回值:
	//   - map[string]interface{}: 一个包含登录操作结果信息的映射。
	//     需要包含IdentityKey 唯一标识；其余键会整体写进令牌载荷（如 sub/aid/ver）。
	Login(username, password, ext string) map[string]interface{}
	// Validate 动态验证用户登录状态。比如作废token。可选
	// 该函数接收一个唯一标识符（如用户ID）作为参数，并返回一个布尔值。
	// 如果用户已登录，则返回true；否则返回false
	// 参数:
	//   - identityVal: 用户唯一标识符内容，用于验证用户登录状态。
	// 返回值:
	//   - bool: 如果用户已登录，则返回true；否则返回false
	Validate() func(identityVal string) bool

	// ResponseWrapper 响应包装。可选
	ResponseWrapper(ctx context.Context, c *app.RequestContext, code int, message map[string]interface{})

	// UnauthorizedWrapper 401错误包装。可选
	UnauthorizedWrapper(ctx context.Context, c *app.RequestContext, code int, message string)
}

// JWTAuthenticator 是可选扩展点：实现它以后，默认的「用户名 + 密码」绑定被整体接管，
// 邮箱 + 验证码、手机 + 短信、第三方回调等登录方式都可以在这里按请求内容分派。
// 返回的 map 会整体写进令牌载荷（保留键 exp/orig_iat 除外），必须包含 IdentityKey() 对应的值。
type JWTAuthenticator interface {
	Authenticate(ctx context.Context, c *app.RequestContext) (map[string]interface{}, error)
}

// AuthenticateFunc 是登录用例的函数形状：从请求里取凭证并换回令牌载荷。
// 返回值可以是 map[string]interface{}（载荷），也可以是 LoginResult（载荷 + 本次有效期）。
type AuthenticateFunc func(ctx context.Context, c *app.RequestContext) (interface{}, error)

// LoginResult 是登录回调的可选返回：需要「同一入口、按请求决定令牌时长」时用它。
// TTL <= 0 表示沿用入口或配置的有效期。
type LoginResult struct {
	// Claims 是写进令牌的载荷，必须包含身份键。
	Claims map[string]interface{}
	// TTL 是本次签发的有效期。
	TTL time.Duration
}

// ClaimsBuilder 覆盖令牌载荷组装：data 是登录或签发传入的数据，返回值就是 claims。
type ClaimsBuilder func(data interface{}) jwt.MapClaims

// LoginResponder 自定义登录 / 刷新成功响应：拿到令牌、过期时间与本次载荷后，字段由使用方决定。
type LoginResponder func(ctx context.Context, c *app.RequestContext, token string, expireAt time.Time, claims map[string]interface{})

// JWTLoginResponder 是可选扩展点：JWTUser 自己决定登录 / 刷新成功响应（默认只有 {token, expire}）。
type JWTLoginResponder interface {
	LoginResponse(ctx context.Context, c *app.RequestContext, token string, expireAt time.Time, claims map[string]interface{})
}

// JWTValidator 是可选扩展点：带 ctx 的动态校验。
// Validate() 的签名没有 ctx，需要查库 / 查缓存作废令牌时实现它（否则只能用 Validate）。
type JWTValidator interface {
	ValidateIdentity(ctx context.Context, identityVal string) bool
}

type JWTConf struct {
	Disable    bool          `yaml:"disable"`
	Algorithm  string        `yaml:"algorithm"`
	Realm      string        `yaml:"realm"`
	Timeout    time.Duration `yaml:"timeout"`
	MaxRefresh time.Duration `yaml:"max_refresh"`
	Secret     string        `yaml:"secret"`

	hertzJWT    *jwt.HertzJWTMiddleware
	hertzJWTErr error

	user JWTUser
	Ctx  any

	// claimsBuilder 覆盖载荷组装；nil 时把登录结果 map 原样透传。
	claimsBuilder ClaimsBuilder
	// loginResponder 覆盖登录 / 刷新成功响应；nil 时按 JWTUser 或默认 {token, expire} 输出。
	loginResponder LoginResponder
}

func (j *JWTConf) initialize() {
	if j.Realm == "" {
		j.Realm = "mali zone"
	}
	if j.Timeout == 0 {
		j.Timeout = time.Hour
	}
	if j.MaxRefresh == 0 {
		j.MaxRefresh = time.Hour
	}
	if j.Secret == "" {
		j.Secret = "mali secret key"
	}
}

func (j *JWTConf) ResetUserConfig(u JWTUser) {
	j.initialize()
	if u == nil {
		return
	}
	j.user = u
	j.hertzJWT, j.hertzJWTErr = j.jwtMiddlewareInit()
}

// GetJwtMiddlewareCore 取底层 hertz-jwt 中间件，首次调用时按当前配置初始化。
func (j *JWTConf) GetJwtMiddlewareCore() (*jwt.HertzJWTMiddleware, error) {
	j.initialize()
	if j.hertzJWT == nil {
		j.hertzJWT, j.hertzJWTErr = j.jwtMiddlewareInit()
	}

	return j.hertzJWT, j.hertzJWTErr
}

// SetClaimsBuilder 覆盖载荷组装规则；不设置时按 map 原样透传（丢掉 exp/orig_iat），
// 足以承载 sub/aid/ver 这类自定义字段。只能在启动期设置。
func (j *JWTConf) SetClaimsBuilder(builder ClaimsBuilder) {
	j.claimsBuilder = builder
}

// SetLoginResponder 覆盖登录 / 刷新成功响应：拿到 token、expireAt 与本次载荷后自行决定字段。
// 不设置时依次回落：JWTUser 实现 JWTLoginResponder → ResponseWrapper → 默认 {token, expire}。
// 只能在启动期设置。
func (j *JWTConf) SetLoginResponder(responder LoginResponder) {
	j.loginResponder = responder
}

// identityKey 返回身份键：用户配置优先，未配置时回落 id。
func (j *JWTConf) identityKey() string {
	if j.user != nil {
		if key := j.user.IdentityKey(); key != "" {
			return key
		}
	}

	return defaultIdentityKey
}

// buildClaims 把登录结果整理成令牌载荷：默认整体透传 map，设置了 ClaimsBuilder 时以构建结果为准。
func (j *JWTConf) buildClaims(data interface{}) jwt.MapClaims {
	if result, ok := data.(LoginResult); ok {
		data = result.Claims
	}
	if j.claimsBuilder != nil {
		return j.claimsBuilder(data)
	}

	claims := jwt.MapClaims{}

	var payload map[string]interface{}
	switch v := data.(type) {
	case map[string]interface{}:
		payload = v
	case jwt.MapClaims:
		payload = map[string]interface{}(v)
	}

	for key, val := range payload {
		if key == claimExp || key == claimOrigIat {
			continue
		}
		claims[key] = val
	}

	return claims
}

// hasIdentity 判断载荷是否带有效身份，签发与登录前用它兜住「忘记带身份键」。
func (j *JWTConf) hasIdentity(claims jwt.MapClaims) bool {
	_, ok := ClaimString(claims, j.identityKey())

	return ok
}

// Claims 取当前请求令牌的完整载荷；没有经过中间件时返回空载荷。
func (j *JWTConf) Claims(ctx context.Context, c *app.RequestContext) jwt.MapClaims {
	return jwt.ExtractClaims(ctx, c)
}

// Identity 取当前请求的身份值（IdentityKey 对应的载荷值）；数字型身份会转成字符串。
func (j *JWTConf) Identity(ctx context.Context, c *app.RequestContext) (string, bool) {
	return ClaimString(j.Claims(ctx, c), j.identityKey())
}

// ClaimString 把载荷值转成字符串，兼容 JSON 解码后的 float64、json.Number 与各整型。
func ClaimString(claims map[string]interface{}, key string) (string, bool) {
	return claimString(claims[key])
}

// ClaimInt64 把载荷值读成整数，兼容 JSON 解码后的 float64、json.Number 与字符串数字。
func ClaimInt64(claims map[string]interface{}, key string) (int64, bool) {
	switch v := claims[key].(type) {
	case json.Number:
		n, err := v.Int64()
		return n, err == nil
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint64:
		return int64(v), true
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		return n, err == nil
	}

	return 0, false
}

func claimString(value interface{}) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, v != ""
	case json.Number:
		return v.String(), true
	case float64:
		// JSON 数字解出来是 float64，整数值按整数输出，避免 42 变成 "42.000000"。
		if v == math.Trunc(v) && math.Abs(v) <= 1<<53 {
			return strconv.FormatInt(int64(v), 10), true
		}
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), true
	case int:
		return strconv.Itoa(v), true
	case int32:
		return strconv.FormatInt(int64(v), 10), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case uint:
		return strconv.FormatUint(uint64(v), 10), true
	case uint64:
		return strconv.FormatUint(v, 10), true
	case bool:
		return strconv.FormatBool(v), true
	}

	return "", false
}

func (j *JWTConf) jwtMiddlewareInit() (*jwt.HertzJWTMiddleware, error) {
	identityKey := j.identityKey()

	mw, err := jwt.New(&jwt.HertzJWTMiddleware{
		Realm:            j.Realm,
		SigningAlgorithm: j.Algorithm,
		Key:              []byte(j.Secret),
		Timeout:          j.Timeout,
		MaxRefresh:       j.MaxRefresh,
		IdentityKey:      identityKey,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			// data: Authenticator() / TokenGenerator() 传入的登录结果
			return j.buildClaims(data)
		},
		IdentityHandler: func(ctx context.Context, c *app.RequestContext) interface{} {
			claims := jwt.ExtractClaims(ctx, c)
			if identity, ok := ClaimString(claims, j.identityKey()); ok {
				return identity
			}

			return nil
		},
		Authenticator: func(ctx context.Context, c *app.RequestContext) (interface{}, error) {
			// 自定义登录方式优先：邮箱+验证码、手机+短信、第三方回调由业务实现 JWTAuthenticator。
			if custom, ok := j.user.(JWTAuthenticator); ok {
				return j.authenticate(func(ctx context.Context, c *app.RequestContext) (interface{}, error) {
					return custom.Authenticate(ctx, c)
				}, ctx, c)
			}

			if j.user == nil {
				return nil, jwt.ErrMissingAuthenticatorFunc
			}

			var loginVals struct {
				Username string `form:"username,required" json:"username,required"`
				Password string `form:"password,required" json:"password,required"`
				Ext      string `form:"ext" json:"ext"`
			}
			if err := c.BindAndValidate(&loginVals); err != nil {
				return "", jwt.ErrMissingLoginValues
			}

			login := j.user.Login
			return j.authenticate(func(context.Context, *app.RequestContext) (interface{}, error) {
				return login(loginVals.Username, loginVals.Password, loginVals.Ext), nil
			}, ctx, c)
		},
		Authorizator: func(data interface{}, ctx context.Context, c *app.RequestContext) bool {
			// data: IdentityHandler()
			v, ok := data.(string)
			if !ok || v == "" {
				// 令牌里没有有效身份（缺键或格式不对）一律拒绝，不能放行。
				return false
			}

			// 此处引入动态用户校验：JWTValidator 带 ctx（可查库/缓存），Validate 兼容老实现。
			if j.user != nil {
				if validator, hasValidator := j.user.(JWTValidator); hasValidator {
					return validator.ValidateIdentity(ctx, v)
				}
				if validate := j.user.Validate(); validate != nil {
					return validate(v)
				}

				// 方便无侵入参数绑定
				if key := j.identityKey(); key != "" {
					c.QueryArgs().Add("__"+key+"__", v)
				}
			}

			return true
		},
		Unauthorized: j.unauthorized,
		LoginResponse: func(ctx context.Context, c *app.RequestContext, code int, token string, expire time.Time) {
			j.respondToken(ctx, c, code, token, expire)
		},
		LogoutResponse: func(ctx context.Context, c *app.RequestContext, code int) {
			if j.user != nil {
				j.user.ResponseWrapper(ctx, c, code, nil)
			}
		},
		RefreshResponse: func(ctx context.Context, c *app.RequestContext, code int, token string, expire time.Time) {
			j.respondToken(ctx, c, code, token, expire)
		},
	})
	if err != nil {
		j.hertzJWTErr = fmt.Errorf("jwt.New() Error: %w", err)
		return nil, j.hertzJWTErr
	}

	j.hertzJWTErr = nil
	return mw, nil
}

// authenticate 统一登录收敛：凭证函数给出载荷后校验身份键，缺身份直接失败，
// 避免签出一个此后每个请求都会被拒的令牌。
func (j *JWTConf) authenticate(fn AuthenticateFunc, ctx context.Context, c *app.RequestContext) (interface{}, error) {
	data, err := fn(ctx, c)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, jwt.ErrFailedAuthentication
	}
	if !j.hasIdentity(j.buildClaims(data)) {
		return nil, fmt.Errorf("jwt: 载荷缺少身份键 %q", j.identityKey())
	}

	return data, nil
}

// respondToken 输出登录 / 刷新成功响应：配置级 Responder > JWTUser 自定义响应 > ResponseWrapper > 默认 JSON。
func (j *JWTConf) respondToken(ctx context.Context, c *app.RequestContext, code int, token string, expireAt time.Time) {
	claims := j.claimsOfToken(token)

	if j.loginResponder != nil {
		j.loginResponder(ctx, c, token, expireAt, claims)
		return
	}
	if responder, ok := j.user.(JWTLoginResponder); ok {
		responder.LoginResponse(ctx, c, token, expireAt, claims)
		return
	}

	raw := map[string]interface{}{
		"token":  token,
		"expire": expireAt.Format(time.DateTime),
	}
	if j.user != nil {
		j.user.ResponseWrapper(ctx, c, code, raw)
		return
	}

	c.JSON(code, raw)
}

// claimsOfToken 解析刚签发的令牌拿载荷：hertz-jwt 的登录回调只给 token，不给 claims。
func (j *JWTConf) claimsOfToken(token string) map[string]interface{} {
	parsed, err := j.ParseTokenString(token)
	if err != nil {
		return map[string]interface{}{}
	}

	claims, ok := parsed.Claims.(rawjwt.MapClaims)
	if !ok {
		return map[string]interface{}{}
	}

	return claims
}

func (j *JWTConf) unauthorized(ctx context.Context, c *app.RequestContext, code int, message string) {
	if j.user != nil {
		j.user.UnauthorizedWrapper(ctx, c, code, message)
		return
	}

	c.JSON(code, map[string]interface{}{
		"code":    code,
		"message": message,
	})
}

func (j *JWTConf) failedHandler(err error) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		j.unauthorized(ctx, c, http.StatusInternalServerError, err.Error())
	}
}

// SetTimeout 修改共享中间件的有效期：只给启动期用。
// 运行期按次指定有效期请用 TokenGeneratorWithTTL，否则会改到并发中的其它请求。
func (j *JWTConf) SetTimeout(t time.Duration) {
	j.Timeout = t
	if j.hertzJWT != nil {
		j.hertzJWT.Timeout = t
	}
}

// SetMaxRefresh 修改共享中间件的最大刷新窗口：只给启动期用。
func (j *JWTConf) SetMaxRefresh(t time.Duration) {
	j.MaxRefresh = t
	if j.hertzJWT != nil {
		j.hertzJWT.MaxRefresh = t
	}
}

func (j *JWTConf) SetSecret(s string) {
	j.Secret = s
	if j.hertzJWT != nil {
		j.hertzJWT.Key = []byte(s)
	}
}

// TokenGenerator 用配置里的有效期签发令牌；payload 为登录结果（map 透传为载荷，见 ClaimsBuilder）。
func (j *JWTConf) TokenGenerator(payload interface{}) (string, time.Time, error) {
	return j.tokenGenerator(payload, 0)
}

// TokenGeneratorWithTTL 用指定有效期签发令牌，不修改共享中间件（并发安全）。
// 后台长令牌、移动端短令牌这类「同一套密钥、不同时长」的签发用它，别用 SetTimeout 再签发。
func (j *JWTConf) TokenGeneratorWithTTL(payload interface{}, ttl time.Duration) (string, time.Time, error) {
	return j.tokenGenerator(payload, ttl)
}

func (j *JWTConf) tokenGenerator(payload interface{}, ttl time.Duration) (string, time.Time, error) {
	mw, err := j.GetJwtMiddlewareCore()
	if err != nil {
		return "", time.Time{}, err
	}
	if !j.hasIdentity(j.buildClaims(payload)) {
		return "", time.Time{}, fmt.Errorf("jwt: 载荷缺少身份键 %q", j.identityKey())
	}

	if ttl <= 0 {
		return mw.TokenGenerator(payload)
	}

	// 复制中间件改有效期：hertz-jwt 的 TokenGenerator 只读 TimeoutFunc，不需要重建引擎。
	scoped := *mw
	scoped.Timeout = ttl
	scoped.TimeoutFunc = func(rawjwt.MapClaims) time.Duration {
		return ttl
	}

	return scoped.TokenGenerator(payload)
}

func (j *JWTConf) ParseTokenString(tokenStr string) (*rawjwt.Token, error) {
	mw, err := j.GetJwtMiddlewareCore()
	if err != nil {
		return nil, err
	}

	return mw.ParseTokenString(tokenStr)
}

func (j *JWTConf) CheckIfTokenExpire(ctx context.Context, c *app.RequestContext) (rawjwt.MapClaims, error) {
	mw, err := j.GetJwtMiddlewareCore()
	if err != nil {
		return nil, err
	}

	return mw.CheckIfTokenExpire(ctx, c)
}

func (j *JWTConf) CheckIfTokenExpireByTokenString(tokenStr string) (rawjwt.MapClaims, error) {
	mw, err := j.GetJwtMiddlewareCore()
	if err != nil {
		return nil, err
	}

	token, err := mw.ParseTokenString(tokenStr)
	if err != nil {
		var validationErr *rawjwt.ValidationError
		ok := errors.As(err, &validationErr)
		if !ok || validationErr.Errors != rawjwt.ValidationErrorExpired {
			return nil, err
		}
	}
	if token == nil {
		return nil, jwt.ErrExpiredToken
	}

	claims := token.Claims.(rawjwt.MapClaims)

	// 检查令牌是否超过了最大刷新时间限制
	origIat, ok := ClaimInt64(claims, claimOrigIat)
	if !ok {
		return nil, jwt.ErrExpiredToken
	}
	if origIat < mw.TimeFunc().Add(-mw.MaxRefresh).Unix() {
		return nil, jwt.ErrExpiredToken
	}

	// 检查令牌是否超过了过期时间限制
	if _, exists := claims[claimExp]; !exists {
		return nil, jwt.ErrMissingExpField
	}
	exp, ok := ClaimInt64(claims, claimExp)
	if !ok {
		return nil, jwt.ErrWrongFormatOfExp
	}
	if exp < mw.TimeFunc().Unix() {
		return nil, jwt.ErrExpiredToken
	}

	return claims, nil
}

func (j *JWTConf) MiddlewareFunc() app.HandlerFunc {
	if j.Disable {
		return func(ctx context.Context, c *app.RequestContext) {
			c.Next(ctx)
		}
	}

	mw, err := j.GetJwtMiddlewareCore()
	if err != nil {
		return j.failedHandler(err)
	}

	return mw.MiddlewareFunc()
}

// LoginHandler 默认登录入口：用户名 + 密码（或 JWTUser 实现了 JWTAuthenticator 时由它接管）。
func (j *JWTConf) LoginHandler() app.HandlerFunc {
	mw, err := j.GetJwtMiddlewareCore()
	if err != nil {
		return j.failedHandler(err)
	}

	return mw.LoginHandler
}

// LoginHandlerWith 生成一个自定义登录入口：用 fn 校验凭证并返回载荷，
// 其余行为（载荷规则、有效期、Cookie、登录响应与 401 包装）与默认登录完全一致。
// 邮箱+验证码、手机+短信、第三方回调等各自的路由都挂它，令牌形态保持一致。
func (j *JWTConf) LoginHandlerWith(fn AuthenticateFunc) app.HandlerFunc {
	return j.loginHandler(fn, 0)
}

// LoginHandlerWithTTL 同 LoginHandlerWith，但用指定有效期签发（不改共享中间件）。
// 不同登录入口的令牌时长策略不同（如后台 3 天、移动端 20 小时）时用它。
func (j *JWTConf) LoginHandlerWithTTL(fn AuthenticateFunc, ttl time.Duration) app.HandlerFunc {
	return j.loginHandler(fn, ttl)
}

func (j *JWTConf) loginHandler(fn AuthenticateFunc, ttl time.Duration) app.HandlerFunc {
	mw, err := j.GetJwtMiddlewareCore()
	if err != nil {
		return j.failedHandler(err)
	}

	base := *mw
	if ttl > 0 {
		base.Timeout = ttl
		base.TimeoutFunc = func(rawjwt.MapClaims) time.Duration {
			return ttl
		}
	}

	return func(ctx context.Context, c *app.RequestContext) {
		// 先校验凭证：拿到载荷后才能决定本次有效期（LoginResult.TTL），也避免签出缺身份的令牌。
		data, authErr := j.authenticate(fn, ctx, c)
		if authErr != nil {
			base.Unauthorized(ctx, c, http.StatusUnauthorized, base.HTTPStatusMessageFunc(authErr, ctx, c))
			return
		}

		// 只替换 Authenticator：载荷合并、过期时间、Cookie、响应包装都复用 hertz-jwt 的 LoginHandler 实现。
		scoped := base
		if result, ok := data.(LoginResult); ok && result.TTL > 0 {
			scoped.Timeout = result.TTL
			scoped.TimeoutFunc = func(rawjwt.MapClaims) time.Duration {
				return result.TTL
			}
		}
		scoped.Authenticator = func(context.Context, *app.RequestContext) (interface{}, error) {
			return data, nil
		}

		scoped.LoginHandler(ctx, c)
	}
}

func (j *JWTConf) RefreshHandler() app.HandlerFunc {
	mw, err := j.GetJwtMiddlewareCore()
	if err != nil {
		return j.failedHandler(err)
	}

	return mw.RefreshHandler
}

func (j *JWTConf) LogoutHandler() app.HandlerFunc {
	mw, err := j.GetJwtMiddlewareCore()
	if err != nil {
		return j.failedHandler(err)
	}

	return mw.LogoutHandler
}
