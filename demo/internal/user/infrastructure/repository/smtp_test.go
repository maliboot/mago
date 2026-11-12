package repository

import (
	"bufio"
	"context"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maliboot/mago/config"
)

// fakeSMTP 是一个最小 SMTP 服务器：完成握手、收下报文，供适配器测试断言内容。
type fakeSMTP struct {
	addr string
	mu   sync.Mutex
	// data 保存最近一次收到的报文正文。
	data string
	// rcpt 保存最近一次的信封收件人。
	rcpt string
}

// newFakeSMTP 启动假服务器并在测试结束时关闭。
func newFakeSMTP(t *testing.T) *fakeSMTP {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}

	server := &fakeSMTP{addr: listener.Addr().String()}
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			server.serve(conn)
		}
	}()

	t.Cleanup(func() {
		_ = listener.Close()
		<-done
	})

	return server
}

// serve 处理一次 SMTP 会话。
func (f *fakeSMTP) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	write := func(line string) {
		_, _ = writer.WriteString(line + "\r\n")
		_ = writer.Flush()
	}

	write("220 fake-smtp")

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))

		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			write("250-fake-smtp")
			write("250 8BITMIME")
		case strings.HasPrefix(cmd, "MAIL FROM"):
			write("250 OK")
		case strings.HasPrefix(cmd, "RCPT TO"):
			f.mu.Lock()
			f.rcpt = strings.Trim(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "RCPT TO:")), "<>")
			f.mu.Unlock()
			write("250 OK")
		case cmd == "DATA":
			write("354 End data with <CR><LF>.<CR><LF>")
			var body strings.Builder
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				if strings.TrimRight(dataLine, "\r\n") == "." {
					break
				}
				body.WriteString(dataLine)
			}
			f.mu.Lock()
			f.data = body.String()
			f.mu.Unlock()
			write("250 Queued")
		case cmd == "QUIT":
			write("221 Bye")
			return
		default:
			write("250 OK")
		}
	}
}

func (f *fakeSMTP) message() (rcpt, data string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.rcpt, f.data
}

func TestSMTPSendCode(t *testing.T) {
	server := newFakeSMTP(t)

	host, port, err := net.SplitHostPort(server.addr)
	if err != nil {
		t.Fatalf("拆分地址失败: %v", err)
	}

	portNum, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("解析端口失败: %v", err)
	}
	smtpConf := &config.SMTP{Host: host, Port: portNum, From: "no-reply@example.com"}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = NewSMTP(smtpConf).SendCode(ctx, "user@example.com", "123456"); err != nil {
		t.Fatalf("SendCode: %v", err)
	}

	rcpt, data := server.message()
	if rcpt != "user@example.com" {
		t.Fatalf("信封收件人 = %q, 期望 user@example.com", rcpt)
	}
	for _, want := range []string{"123456", "Subject:", "Content-Type: text/plain; charset=UTF-8"} {
		if !strings.Contains(data, want) {
			t.Fatalf("报文缺少 %q:\n%s", want, data)
		}
	}
}

func TestSMTPSendCodeWithoutConfig(t *testing.T) {
	if err := NewSMTP(nil).SendCode(context.Background(), "user@example.com", "123456"); err == nil {
		t.Fatal("未配置 smtp 时应报错")
	}
}
