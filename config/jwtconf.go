package config

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/jwt"
)

type JWTUser interface {
	// IdentityKey 唯一标识，如id。必须
	IdentityKey() string
	// Login 执行用户登录操作。当需要自定义[jwt.LoginHandler]登录逻辑，则可以重写该函数。否则是是可选的。
	// 该函数接收三个参数：username（用户名），password（密码），ext（扩展信息）。
	// 它返回一个映射，其中包含登录结果的相关信息。
	// 参数:
	//   - username: 用户名字符串，用于标识用户。
	//   - password: 密码字符串，用于验证用户身份。
	//   - ext: 扩展信息字符串，可用于额外的验证或跟踪信息。
	// 返回值:
	//   - map[string]interface{}: 一个包含登录操作结果信息的映射。
	//     需要包含IdentityKey 唯一标识
	//     其它具体内容取决于登录逻辑和结果。
	Login(username, password, ext string) map[string]interface{}
	// Validate 动态验证用户登录状态。比如作废token。可选
	// 该函数接收一个唯一标识符（如用户ID）作为参数，并返回一个布尔值。
	// 如果用户已登录，则返回true；否则返回false。
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

type JWTConf struct {
	Algorithm  string        `yaml:"algorithm"`
	Realm      string        `yaml:"realm"`
	Timeout    time.Duration `yaml:"timeout"`
	MaxRefresh time.Duration `yaml:"max_refresh"`
	Secret     string        `yaml:"secret"`

	hertzJWT    *jwt.HertzJWTMiddleware
	hertzJWTErr error

	user JWTUser
	Ctx  any
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

func (j *JWTConf) GetJwtMiddlewareCore() (*jwt.HertzJWTMiddleware, error) {
	if j.hertzJWT == nil {
		j.hertzJWT, j.hertzJWTErr = j.jwtMiddlewareInit()
	}

	j.initialize()
	return j.hertzJWT, j.hertzJWTErr
}

func (j *JWTConf) jwtMiddlewareInit() (*jwt.HertzJWTMiddleware, error) {
	identityKey := "id"
	if j.user != nil {
		identityKey = j.user.IdentityKey()
	}

	j.hertzJWT, j.hertzJWTErr = jwt.New(&jwt.HertzJWTMiddleware{
		Realm:            j.Realm,
		SigningAlgorithm: j.Algorithm,
		Key:              []byte(j.Secret),
		Timeout:          j.Timeout,
		MaxRefresh:       j.MaxRefresh,
		IdentityKey:      identityKey,
		PayloadFunc: func(data interface{}) jwt.MapClaims {
			// data: Authenticator()返回数据
			if j.user == nil {
				return jwt.MapClaims{}
			}

			identityKey = j.user.IdentityKey()
			if identityVal, ok := data.(map[string]interface{})[identityKey]; ok {
				return jwt.MapClaims{
					identityKey: identityVal,
				}
			}
			return jwt.MapClaims{}
		},
		IdentityHandler: func(ctx context.Context, c *app.RequestContext) interface{} {
			claims := jwt.ExtractClaims(ctx, c)
			return claims[identityKey].(string)
		},
		Authenticator: func(ctx context.Context, c *app.RequestContext) (interface{}, error) {
			var loginVals struct {
				Username string `form:"username,required" json:"username,required"`
				Password string `form:"password,required" json:"password,required"`
				Ext      string `form:"ext" json:"ext"`
			}
			if err := c.BindAndValidate(&loginVals); err != nil {
				return "", jwt.ErrMissingLoginValues
			}

			userData := j.user.Login(loginVals.Username, loginVals.Password, loginVals.Ext)
			if userData != nil {
				return userData, nil
			}

			return nil, jwt.ErrFailedAuthentication
		},
		Authorizator: func(data interface{}, ctx context.Context, c *app.RequestContext) bool {
			// data: IdentityHandler()
			// 此处引入动态用户校验
			v, ok := data.(string)
			if ok && j.user != nil && j.user.Validate() != nil {
				return (j.user.Validate())(v)
			}

			// 方便无侵入参数绑定
			if j.user != nil && j.user.IdentityKey() != "" {
				c.QueryArgs().Add("__"+j.user.IdentityKey()+"__", v)
			}
			return true
		},
		Unauthorized: func(ctx context.Context, c *app.RequestContext, code int, message string) {
			if j.user != nil {
				j.user.UnauthorizedWrapper(ctx, c, code, message)
			} else {
				c.JSON(code, map[string]interface{}{
					"code":    code,
					"message": message,
				})
			}
		},
		LoginResponse: func(ctx context.Context, c *app.RequestContext, code int, token string, expire time.Time) {
			raw := map[string]interface{}{
				"token":  token,
				"expire": expire.Format(time.DateTime),
			}
			if j.user != nil {
				j.user.ResponseWrapper(ctx, c, code, raw)
			} else {
				c.JSON(code, raw)
			}
		},
		LogoutResponse: func(ctx context.Context, c *app.RequestContext, code int) {
			if j.user != nil {
				j.user.ResponseWrapper(ctx, c, code, nil)
			}
		},
		RefreshResponse: func(ctx context.Context, c *app.RequestContext, code int, token string, expire time.Time) {
			raw := map[string]interface{}{
				"token":  token,
				"expire": expire.Format(time.DateTime),
			}
			if j.user != nil {
				j.user.ResponseWrapper(ctx, c, code, raw)
			} else {
				c.JSON(code, raw)
			}
		},
	})
	if j.hertzJWTErr != nil {
		return nil, fmt.Errorf("jwt.New() Error:" + j.hertzJWTErr.Error())
	}

	errInit := j.hertzJWT.MiddlewareInit()
	if errInit != nil {
		return nil, fmt.Errorf("jwt.MiddlewareInit() Error:" + errInit.Error())
	}

	return j.hertzJWT, nil
}

func (j *JWTConf) SetTimeout(t time.Duration) {
	if j.hertzJWT != nil {
		j.hertzJWT.Timeout = t
	}
}

func (j *JWTConf) SetMaxRefresh(t time.Duration) {
	if j.hertzJWT != nil {
		j.hertzJWT.MaxRefresh = t
	}
}

func (j *JWTConf) SetSecret(s string) {
	if j.hertzJWT != nil {
		j.hertzJWT.Key = []byte(s)
	}
}

func (j *JWTConf) TokenGenerator(payload interface{}) (string, time.Time, error) {
	return j.hertzJWT.TokenGenerator(payload)
}

func (j *JWTConf) CheckIfTokenExpire(ctx context.Context, c *app.RequestContext) error {
	_, err := j.hertzJWT.CheckIfTokenExpire(ctx, c)
	return err
}

func (j *JWTConf) MiddlewareFunc() app.HandlerFunc {
	return j.hertzJWT.MiddlewareFunc()
}

func (j *JWTConf) LoginHandler() app.HandlerFunc {
	return j.hertzJWT.LoginHandler
}

func (j *JWTConf) RefreshHandler() app.HandlerFunc {
	return j.hertzJWT.RefreshHandler
}

func (j *JWTConf) LogoutHandler() app.HandlerFunc {
	return j.hertzJWT.LogoutHandler
}
