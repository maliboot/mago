//go:build smtplive

// 真实 SMTP 联调：只在显式指定 tag 时运行。
//
//	cd demo/internal/user/infrastructure/repository && go test -tags smtplive -run TestSMTPLive -v -timeout 2m
//
// 连接参数与测试收件箱从仓库根 runtime/smtp.txt 读取（KEY=VALUE），代码里不落任何口令。
package repository

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/maliboot/mago/config"
)

// smtpConfPath 是凭据文件相对包目录的位置（包在 demo/internal/user/infrastructure/repository）。
const smtpConfPath = "../../../../../runtime/smtp.txt"

// liveSMTPConf 是联调所需的参数。
type liveSMTPConf struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	SSL      bool
	To       string
}

// loadLiveSMTPConf 解析 KEY=VALUE 形式的配置文件，并取出注释里的测试收件箱。
func loadLiveSMTPConf(t *testing.T) liveSMTPConf {
	t.Helper()

	bs, err := os.ReadFile(smtpConfPath)
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", smtpConfPath, err)
	}

	var conf liveSMTPConf
	for _, line := range strings.Split(string(bs), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			// 支持「测试邮箱为: xxx」这类注释行
			if _, after, ok := strings.Cut(line, "测试邮箱"); ok {
				conf.To = strings.TrimSpace(strings.TrimLeft(after, "为:： "))
			}
			continue
		}

		// 说明行（无 =）里可能带测试收件箱，如「测试邮箱为: xxx@yyy」
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			if _, after, has := strings.Cut(line, "测试邮箱"); has {
				conf.To = strings.TrimSpace(strings.TrimLeft(after, "为:： "))
			}
			continue
		}

		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		switch key {
		case "SMTP_HOST":
			conf.SSL = strings.HasPrefix(value, "ssl://")
			conf.Host = strings.TrimPrefix(strings.TrimPrefix(value, "ssl://"), "smtp://")
		case "SMTP_PORT":
			conf.Port, _ = strconv.Atoi(value)
		case "SMTP_USER":
			conf.User = value
		case "SMTP_PASSWORD":
			conf.Password = value
		case "SMTP_FROM":
			conf.From = value
		case "SMTP_TLS":
			if strings.EqualFold(value, "true") {
				conf.SSL = true
			}
		}
	}

	if conf.Host == "" || conf.User == "" || conf.Password == "" {
		t.Fatalf("配置缺少 SMTP_HOST / SMTP_USER / SMTP_PASSWORD")
	}
	if conf.To == "" {
		conf.To = conf.From
	}

	return conf
}

func TestSMTPLive(t *testing.T) {
	conf := loadLiveSMTPConf(t)
	t.Logf("目标服务器: %s:%d (ssl=%v)", conf.Host, conf.Port, conf.SSL)
	t.Logf("发件人: %s → 收件人: %s", conf.From, conf.To)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	adapter := NewSMTP(&config.SMTP{
		Host:     conf.Host,
		Port:     conf.Port,
		Username: conf.User,
		Password: conf.Password,
		From:     conf.From,
		SSL:      conf.SSL,
		Enable:   true,
	})

	const code = "680413"
	if err := adapter.SendCode(ctx, conf.To, code); err != nil {
		t.Fatalf("SendCode: %v", err)
	}

	t.Logf("已投递：%s 收到验证码 %s（请在收件箱与实际邮件核对）", conf.To, code)
}
