// Package mail 存放邮件类外部调用的请求内容（DO），按通用协议分类（不是按业务功能命名）：
// 只描述报文本身，不建立连接、不发邮件，具体调用逻辑在 infrastructure/repository/smtp.go。
package mail

// VerifyCode 是一封验证码邮件的请求内容。
type VerifyCode struct {
	// From 是信封发件人。
	From string
	// To 是收件人邮箱。
	To string
	// Subject 是邮件主题（非 ASCII 字符由发送方编码）。
	Subject string
	// Body 是正文纯文本。
	Body string
}
