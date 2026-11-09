package autoloadx

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/maliboot/mago/config"
	"github.com/maliboot/mago/logx"
)

func AccessLog(conf *config.Conf) app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()
		c.Next(ctx)
		end := time.Now()
		latency := end.Sub(start).Microseconds()

		content := c.Request.Body()
		if strings.HasPrefix(string(c.ContentType()), "multipart/form-data") {
			content = []byte("")
		}

		rspErr := []byte("")
		if c.Response.StatusCode() != http.StatusOK {
			rspErr = c.Response.Body()
		}

		requestId := c.Response.Header.Get("X-Request-ID")

		logx.GroupInfof("access", "status=%d cost=%d method=%s full_path=%s client_ip=%s host=%s requestId=%s body=%s error=%s",
			c.Response.StatusCode(), latency,
			c.Request.Header.Method(), c.Request.URI().PathOriginal(), c.ClientIP(), c.Request.Host(), requestId, content, rspErr,
		)
	}
}
