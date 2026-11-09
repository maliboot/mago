package emqx

import (
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

type MQTTLog struct {
	level      hlog.Level
	prefixWord string
}

func (l *MQTTLog) Println(v ...interface{}) {
	switch l.level {
	case hlog.LevelDebug:
		hlog.Debug(l.prefixWord+"debug", v)
		break
	case hlog.LevelInfo:
		hlog.Info(l.prefixWord+"info", v)
		break
	case hlog.LevelError:
		hlog.Error(l.prefixWord+"error", v)
		break
	case hlog.LevelFatal:
		hlog.Fatal(l.prefixWord+"fatal", v)
		break
	case hlog.LevelWarn:
		hlog.Warn(l.prefixWord+"warn", v)
		break
	default:
		hlog.Trace(l.prefixWord, v)
	}
}

func (l *MQTTLog) Printf(format string, v ...interface{}) {
	format = l.prefixWord + format
	switch l.level {
	case hlog.LevelDebug:
		hlog.Debugf(format, v)
		break
	case hlog.LevelInfo:
		hlog.Infof(format, v)
		break
	case hlog.LevelError:
		hlog.Errorf(format, v)
		break
	case hlog.LevelFatal:
		hlog.Fatalf(format, v)
		break
	case hlog.LevelWarn:
		hlog.Warnf(format, v)
		break
	default:
		hlog.Tracef(format, v)
	}
}

func NewMQTTLog(l hlog.Level) *MQTTLog {
	return &MQTTLog{level: l, prefixWord: "mqtt//"}
}
