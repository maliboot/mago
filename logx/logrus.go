package logx

import (
	"fmt"
	"io"
	"path"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	hertzlogrus "github.com/hertz-contrib/logger/logrus"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logRusIns *LogRus
)

type LogRus struct {
	appEnv       string
	LogDir       string
	defaultGroup string

	rawLogrusLogger *logrus.Logger

	date        string     // 当前日期，格式如 "20060102"
	mutex       sync.Mutex // 保护 writer 更新
	writers     sync.Map
	filenameMap map[logrus.Level]string
}

func NewLogRus(appEnv, LogDir string) *LogRus {
	if logRusIns != nil {
		return logRusIns
	}
	logRusIns = &LogRus{
		appEnv: appEnv,
		LogDir: LogDir,
	}
	logRusIns.initialize()
	return logRusIns
}

func (l *LogRus) initialize() {
	if l.defaultGroup == "" {
		l.defaultGroup = "app"
	}

	// 日志日期格式
	l.date = time.Now().Format("20060102")
	// 级别前缀
	l.filenameMap = map[logrus.Level]string{
		logrus.TraceLevel: "trace",
		logrus.DebugLevel: "debug",
		logrus.InfoLevel:  "info",
		logrus.WarnLevel:  "warn",
		logrus.ErrorLevel: "error",
		logrus.FatalLevel: "fatal",
	}

	// 创建 logrus Logger
	l.rawLogrusLogger = logrus.New()
	if l.appEnv != "production" {
		l.rawLogrusLogger.SetLevel(logrus.TraceLevel)
	} else {
		l.rawLogrusLogger.SetLevel(logrus.InfoLevel)
	}
	l.rawLogrusLogger.AddHook(l) // 添加 Hook

	if l.appEnv == "dev" {
		return
	}

	// 禁止stdio/stderr
	l.rawLogrusLogger.SetOutput(io.Discard)

	// 使用 Hertz 的 logrus 适配器注入 hlog
	hlogLogger := hertzlogrus.NewLogger(hertzlogrus.WithLogger(l.rawLogrusLogger))
	hlog.SetLogger(hlogLogger)
}

func (l *LogRus) getWriterKey(group string, level logrus.Level) string {
	levelStr := l.filenameMap[level]
	if levelStr == "" {
		levelStr = "info" // 默认 fallback
	}
	return fmt.Sprintf("%s-%s", group, levelStr)
}

func (l *LogRus) getOrCreateWriter(group string, level logrus.Level) *lumberjack.Logger {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// 检查当前日期
	currentDate := time.Now().Format("20060102")
	if l.date != currentDate {
		// 日期变化，重置所有 writer
		l.date = currentDate
		l.writers.Range(func(k, w interface{}) bool {
			if w == nil {
				return true
			}

			err := w.(*lumberjack.Logger).Close()
			if err != nil {
				fmt.Printf("Rotated log file failed: [%s]", k)
				return true
			}
			return true
		})
		l.writers.Clear()
	}

	key := l.getWriterKey(group, level)
	if writer, exists := l.writers.Load(key); exists && writer != nil {
		return writer.(*lumberjack.Logger)
	}

	// 创建新 writer
	writer := l.newLumberjackLogger(group, level)
	l.writers.Store(key, writer)
	return writer
}

func (l *LogRus) newLumberjackLogger(group string, level logrus.Level) *lumberjack.Logger {
	prefix := group
	if prefix == "" {
		prefix = l.defaultGroup // 兜底使用全局默认前缀
	}

	filename := path.Join(l.LogDir, fmt.Sprintf("%s-%s-%s.log", prefix, l.filenameMap[level], l.date))
	return &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    20,   // MB
		MaxBackups: 5,    // 最多保留 5 个备份
		MaxAge:     30,   // 保留 30 天
		Compress:   true, // 压缩旧文件
	}
}

func (l *LogRus) Fire(entry *logrus.Entry) error {
	group := l.defaultGroup
	if g, ok := entry.Data["log_group"]; ok {
		if gs, ok := g.(string); ok && gs != "" {
			group = gs
		}
	}

	writer := l.getOrCreateWriter(group, entry.Level)

	// 格式化日志消息（可自定义格式）
	msg := fmt.Sprintf("[%s] [%s] %s: %s\n", entry.Time.Format(time.DateTime), group, entry.Level.String(), entry.Message)
	for k, v := range entry.Data {
		msg += fmt.Sprintf(" %s=%v", k, v)
	}
	_, err := writer.Write([]byte(msg))
	return err
}

func (l *LogRus) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (l *LogRus) WithField(key string, value interface{}) *logrus.Entry {
	return l.rawLogrusLogger.WithField(key, value)
}
