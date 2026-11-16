package conf

import (
	"github.com/maliboot/mago/config"

	"tests/internal/user/application"
	"tests/internal/user/domain"
	"tests/internal/user/infrastructure/repository"
	"tests/internal/user/infrastructure/util"
)

// 本文件只放「无法直接作为 provider 使用」的桥接函数：类型转换、子配置切片、端口绑定。
// 真正的构造逻辑仍在各层构造函数里，wire 只负责按类型把它们串起来（§11.5）。

// provideSMTPConf 只暴露邮件子配置：SMTP 适配器只需要这一段，不接触全局配置。
func provideSMTPConf(c *config.Conf) *config.SMTP {
	return c.SMTP
}

// provideUserGateway 把 *repository.User 绑定到写端口（端口由 domain 定义）。
func provideUserGateway(u *repository.User) domain.UserGateway {
	return u
}

// provideUserQuery 把同一个 *repository.User 绑定到读端口（端口由 application 定义）。
// 两个桥接函数依赖同一个 *repository.User，wire 只会构造一次实例，自然满足「一个 DO 一个 repository」。
func provideUserQuery(u *repository.User) application.UserQuery {
	return u
}

// providePasswordHasher 把无 DO 的纯工具散列实现（util，§18.6）绑定到 application 的技术端口。
func providePasswordHasher(p *util.Password) application.PasswordHasher {
	return p
}

// provideTokenIssuer 把无 DO 的纯工具签发实现（util，§18.6）绑定到 application 的技术端口。
func provideTokenIssuer(t *util.Token) application.TokenIssuer {
	return t
}

// provideCodeSender 把 SMTP 适配器绑定到验证码发送端口（换短信/钉钉实现时只改这一行）。
func provideCodeSender(s *repository.SMTP) application.CodeSender {
	return s
}

// provideTaskQueue 把 queuex（优先 redis）的发布适配器绑定到异步任务端口。
func provideTaskQueue(q *repository.Queue) application.TaskQueue {
	return q
}

// provideVerifyCodeStore 把内存验证码存储绑定到 application 的校验端口（换 Redis 只改这一行）。
func provideVerifyCodeStore(v *repository.VerifyCode) application.VerifyCodeStore {
	return v
}

// provideFrontendJWT 只暴露 frontend 这一组 JWT 配置：登录入口需要它签发令牌（§12.5）。
func provideFrontendJWT(c *config.Conf) *config.JWTConf {
	return c.Server.Http.JWT["frontend"]
}
