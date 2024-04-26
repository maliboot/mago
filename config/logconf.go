package config

import (
	"os"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

type LogConf struct {
	LogDir string `yaml:"log_dir"`
}

// LoggerInit 日志初始化
func (l *LogConf) LoggerInit(isDev bool) (error, func()) {
	emptyFunc := func() {}
	if !isDev && l.LogDir != "" {
		logFile, err := os.OpenFile(l.LogDir+"/output.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err, emptyFunc
		}
		cleanup := func() {
			_ = logFile.Close()
		}
		hlog.SetOutput(logFile)
		return nil, cleanup
	}
	return nil, emptyFunc
}
