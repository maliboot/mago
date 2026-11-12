package config

// SMTP 是邮件发送配置，供 infrastructure 的邮件适配器读取连接参数。
type SMTP struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
	// SSL 为 true 时走隐式 TLS（常见于 465 端口）；否则用 STARTTLS。
	SSL    bool `yaml:"ssl"`
	Enable bool `yaml:"enable"`
}
