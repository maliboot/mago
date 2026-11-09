package logx

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"gorm.io/gorm"
	gormL "gorm.io/gorm/logger"
)

var (
	GromSqliteFilterMap = make(map[string]int)
)

func GromSqliteFilterListRegister(keyword string) {
	if len(GromSqliteFilterMap) > 50 {
		return
	}
	GromSqliteFilterMap[keyword] = 1
}

func GromSqliteFilterListSqlCheck(sql string) bool {
	for s := range GromSqliteFilterMap {
		if strings.Contains(sql, s) {
			return true
		}
	}
	return false
}

type GormLogger struct {
	hertzLogLevel             hlog.Level
	LogLevel                  gormL.LogLevel
	IgnoreRecordNotFoundError bool
	Colorful                  bool
	SlowThreshold             time.Duration
	LogLevelHook              func(gormL.LogLevel)

	logGroup string
}

func NewGormLogger() *GormLogger {
	return &GormLogger{
		logGroup:                  "db",
		LogLevel:                  gormL.Info,
		SlowThreshold:             100 * time.Millisecond,
		IgnoreRecordNotFoundError: true,
	}
}

func (l GormLogger) LogMode(level gormL.LogLevel) gormL.Interface {
	switch level {
	case gormL.Silent:
		l.hertzLogLevel = hlog.LevelFatal
	case gormL.Error:
		l.hertzLogLevel = hlog.LevelError
	case gormL.Warn:
		l.hertzLogLevel = hlog.LevelWarn
	case gormL.Info:
		l.hertzLogLevel = hlog.LevelInfo
	}
	l.LogLevel = level
	hlog.SetLevel(l.hertzLogLevel)
	return l
}

func (l GormLogger) levelHook() {
	if l.LogLevelHook != nil {
		l.LogLevelHook(l.LogLevel)
	}
}

func (l GormLogger) Info(_ context.Context, str string, args ...interface{}) {
	if l.LogLevel < gormL.Info {
		return
	}
	GroupInfof(l.logGroup, str, args...)
	l.levelHook()
}

func (l GormLogger) Warn(_ context.Context, str string, args ...interface{}) {
	if l.LogLevel < gormL.Warn {
		return
	}
	GroupWarnf(l.logGroup, str, args...)
	l.levelHook()
}

func (l GormLogger) Error(_ context.Context, str string, args ...interface{}) {
	if l.LogLevel < gormL.Error {
		return
	}
	GroupErrorf(l.logGroup, str, args...)
	l.levelHook()
}

func (l GormLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= 0 {
		return
	}
	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= gormL.Error && (!l.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)):
		sql, rows := fc()
		GroupErrorf(
			l.logGroup,
			"[trace]err: %s | elapsed: %d | rows: %d | sql: %s",
			err, elapsed, rows, sql,
		)
	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold && l.LogLevel >= gormL.Warn:
		sql, rows := fc()
		GroupWarnf(
			l.logGroup,
			"[trace]elapsed: %d | rows: %d | sql: %s",
			elapsed, rows, sql,
		)
	case l.LogLevel >= gormL.Info:
		sql, rows := fc()
		if !GromSqliteFilterListSqlCheck(sql) {
			GroupTracef(
				l.logGroup,
				"[trace]elapsed: %d | rows: %d | sql: %s",
				elapsed, rows, sql,
			)
		}
	}
	l.levelHook()
}
