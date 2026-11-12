package conf

import (
	"github.com/maliboot/mago/config"

	"tests/internal/user/application"
	"tests/internal/user/domain"
	"tests/internal/user/infrastructure/repository"
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

// providePasswordHasher 把 infrastructure 的散列实现绑定到 application 的技术端口。
func providePasswordHasher(p *repository.Password) application.PasswordHasher {
	return p
}

// provideTokenIssuer 把 infrastructure 的签发实现绑定到 application 的技术端口。
func provideTokenIssuer(t *repository.Token) application.TokenIssuer {
	return t
}

// provideCodeSender 把 SMTP 适配器绑定到验证码发送端口（换短信/钉钉实现时只改这一行）。
func provideCodeSender(s *repository.SMTP) application.CodeSender {
	return s
}
