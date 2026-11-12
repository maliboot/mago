package repository

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/maliboot/mago"
	"github.com/maliboot/mago/config"

	maildo "tests/internal/user/infrastructure/do/mail"
)

// defaultSMTPPort 是未配置端口时的默认值。
const defaultSMTPPort = 25

// SMTP 是验证码邮件的出站适配器：报文内容来自 do/mail，调用逻辑（短连接、鉴权、超时）都在本文件。
type SMTP struct {
	conf *config.SMTP
}

// 非单例：依赖邮件配置，按配置实例化，由组合根创建一次。
func NewSMTP(conf *config.SMTP) *SMTP {
	return &SMTP{conf: conf}
}

// SendCode 组装验证码邮件内容并发送。
func (s *SMTP) SendCode(ctx context.Context, email, code string) error {
	if s.conf == nil || s.conf.Host == "" {
		return mago.NewError(mago.ErrServerError).WithMsg("smtp 未配置")
	}

	msg := &maildo.VerifyCode{
		From:    s.from(),
		To:      email,
		Subject: "登录验证码",
		Body:    fmt.Sprintf("您的验证码是 %s，10 分钟内有效。如非本人操作，请忽略本邮件。", code),
	}

	return s.send(ctx, msg)
}

// send 建立一次短连接完成投递，超时与取消由 ctx 控制。
func (s *SMTP) send(ctx context.Context, msg *maildo.VerifyCode) error {
	if s.conf == nil || s.conf.Host == "" {
		return mago.NewError(mago.ErrServerError).WithMsg("smtp 未配置")
	}

	addr := net.JoinHostPort(s.conf.Host, strconv.Itoa(s.port()))

	client, err := s.dial(ctx, addr)
	if err != nil {
		return fmt.Errorf("smtp 连接失败: %w", err)
	}
	defer func() { _ = client.Close() }()

	if s.conf.Username != "" {
		if err = client.Auth(smtp.PlainAuth("", s.conf.Username, s.conf.Password, s.conf.Host)); err != nil {
			return fmt.Errorf("smtp 鉴权失败: %w", err)
		}
	}

	if err = client.Mail(msg.From); err != nil {
		return fmt.Errorf("smtp MAIL FROM 失败: %w", err)
	}
	if err = client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("smtp RCPT TO 失败: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA 失败: %w", err)
	}
	if err = writeMessage(writer, msg); err != nil {
		return fmt.Errorf("smtp 写入报文失败: %w", err)
	}
	if err = writer.Close(); err != nil {
		return fmt.Errorf("smtp 提交报文失败: %w", err)
	}

	// 关闭连接失败不影响已投递的结果，忽略即可。
	_ = client.Quit()

	return nil
}

// dial 建立连接：SSL 走隐式 TLS，其余交给 STARTTLS（由 smtp.Client 在需要时升级）。
func (s *SMTP) dial(ctx context.Context, addr string) (*smtp.Client, error) {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	if deadline, ok := ctx.Deadline(); ok {
		dialer.Deadline = deadline
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	if s.conf.SSL {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: s.conf.Host, MinVersion: tls.VersionTLS12})
		if err = tlsConn.HandshakeContext(ctx); err != nil {
			_ = conn.Close()
			return nil, err
		}
		conn = tlsConn
	}

	client, err := smtp.NewClient(conn, s.conf.Host)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return client, nil
}

// writeMessage 按 RFC 5322 写出头部与纯文本正文，主题做 MIME 编码。
func writeMessage(w io.Writer, msg *maildo.VerifyCode) error {
	headers := []string{
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"To: " + msg.To,
		"Subject: " + mime.BEncoding.Encode("UTF-8", msg.Subject),
	}

	if _, err := io.WriteString(w, strings.Join(headers, "\r\n")+"\r\n\r\n"+msg.Body); err != nil {
		return err
	}

	return nil
}

func (s *SMTP) port() int {
	if s.conf.Port > 0 {
		return s.conf.Port
	}

	return defaultSMTPPort
}

func (s *SMTP) from() string {
	if s.conf.From != "" {
		return s.conf.From
	}

	return s.conf.Username
}
