package config

import (
	"fmt"
	"os"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	hertzslog "github.com/hertz-contrib/logger/slog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LogConf struct {
	lumberjack.Logger
	LogName string `yaml:"log_name"`
	LogDir  string `yaml:"log_dir"`
	Ctx     any
}

// LoggerInit 日志初始化
func (l *LogConf) LoggerInit(isDev bool) (error, func()) {
	cleanup := func() {}
	if isDev || l.LogDir == "" {
		return nil, cleanup
	}
	if l.LogName == "" {
		l.LogName = "app"
	}

	if err := l.initLogFile(); err != nil {
		return err, cleanup
	}

	logger := hertzslog.NewLogger()
	if l.MaxSize == 0 {
		l.MaxSize = 20
	}
	if l.MaxBackups == 0 {
		l.MaxBackups = 5
	}
	if l.MaxAge == 0 {
		l.MaxAge = 10
	}
	l.Compress = true

	logger.SetOutput(l)
	logger.SetLevel(hlog.LevelDebug)
	hlog.SetLogger(logger)

	go func() {
		for {
			now := time.Now()
			nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			timeUntilMidnight := time.Until(nextMidnight)

			timer := time.NewTimer(timeUntilMidnight)
			<-timer.C

			// 更新日志文件
			err := l.initLogFile()
			if err != nil {
				fmt.Printf("Rotated log file[%s] err1: %+v", l.Filename, err)
				break
			}
			err = l.Rotate()
			if err != nil {
				fmt.Printf("Rotated log file[%s] err2: %+v", l.Filename, err)
				break
			}
			fmt.Println("Rotated log file to:", l.Filename)
		}
	}()
	return nil, cleanup

}

func (l *LogConf) initLogFile() error {
	if l.LogDir[len(l.LogDir)-1] != '/' {
		l.LogDir += "/"
	}

	if err := os.MkdirAll(l.LogDir, 0777); err != nil {
		return err
	}

	l.Filename = fmt.Sprintf("%s%s-%s.log", l.LogDir, l.LogName, time.Now().Format("2006-01-02"))

	//if _, err := os.Stat(l.Filename); err != nil {
	//	if _, err = os.Create(l.Filename); err != nil {
	//		return err
	//	}
	//}
	return nil
}
