package mago

import (
	"context"
	"fmt"
	"time"

	"github.com/maliboot/mago/config"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

type Http struct {
	*server.Hertz
	c *config.Conf
}

func NewHttp(c *config.Conf) *Http {
	return &Http{
		server.Default(server.WithHostPorts(fmt.Sprintf("%s:%d", c.Server.Http.IP, c.Server.Http.Port))),
		c,
	}
}

func (h *Http) Start(_ context.Context) error {
	err := h.Run()
	if err != nil {
		return err
	}
	return nil
}

func (h *Http) Stop(ctx context.Context) error {
	if forceErr, ok := ctx.Value(StopError{}).(error); ok && forceErr != nil {
		// 强制关闭
		hlog.Errorf("Hertz: Receive close signal: error=%v", forceErr)
		if err := h.Close(); err != nil {
			hlog.Errorf("Hertz: Close error=%v", err)
		}
	}
	hlog.SystemLogger().Infof(
		"Server[%s:%d]Begin graceful shutdown, wait at most num=%d seconds...",
		h.c.Server.Http.IP,
		h.c.Server.Http.Port,
		h.GetOptions().ExitWaitTimeout/time.Second,
	)

	ctx, cancel := context.WithTimeout(ctx, h.GetOptions().ExitWaitTimeout/time.Second)
	defer cancel()

	if err := h.Shutdown(ctx); err != nil {
		hlog.SystemLogger().Errorf("Shutdown error=%v", err)
	}
	return nil
}
