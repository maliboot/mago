package helper

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

func Cron(ctx context.Context, cronName string, timeInterval time.Duration, task func(context.Context)) string {
	jobIdBytes := md5.Sum([]byte(cronName))
	jobId := hex.EncodeToString(jobIdBytes[:])

	go func() {
		ticker := time.NewTicker(timeInterval)
		hlog.Infof("定时器[%s]启动", cronName)
		defer func() {
			hlog.Infof("定时器[%s]退出", cronName)
			ticker.Stop()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				func() {
					defer PanicRecover("Cron" + cronName)
					task(ctx)
				}()
			}
		}
	}()

	// jobId
	return jobId
}
