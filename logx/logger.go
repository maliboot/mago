package logx

type Logger interface {
	Trace(v ...interface{})
	Debug(v ...interface{})
	Info(v ...interface{})
	Notice(v ...interface{})
	Warn(v ...interface{})
	Error(v ...interface{})
	Fatal(v ...interface{})
}

type FormatLogger interface {
	Tracef(format string, v ...interface{})
	Debugf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Noticef(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
	Fatalf(format string, v ...interface{})
}

type Log struct {
	group string
}

func NewLog(group string) *Log {
	return &Log{group: group}
}

func (l *Log) Tracef(format string, v ...interface{}) {
	GroupTracef(l.group, format, v...)
}

func (l *Log) Debugf(format string, v ...interface{}) {
	GroupDebugf(l.group, format, v...)
}

func (l *Log) Infof(format string, v ...interface{}) {
	GroupInfof(l.group, format, v...)
}

func (l *Log) Noticef(format string, v ...interface{}) {
	GroupInfof(l.group, format, v...)
}

func (l *Log) Warnf(format string, v ...interface{}) {
	GroupWarnf(l.group, format, v...)
}

func (l *Log) Errorf(format string, v ...interface{}) {
	GroupErrorf(l.group, format, v...)
}

func (l *Log) Fatalf(format string, v ...interface{}) {
	GroupFatalf(l.group, format, v...)
}

func (l *Log) Trace(v ...interface{}) {
	GroupTrace(l.group, v...)
}

func (l *Log) Debug(v ...interface{}) {
	GroupDebug(l.group, v...)
}

func (l *Log) Info(v ...interface{}) {
	GroupInfo(l.group, v...)
}

func (l *Log) Notice(v ...interface{}) {
	GroupInfo(l.group, v...)
}

func (l *Log) Warn(v ...interface{}) {
	GroupWarn(l.group, v...)
}

func (l *Log) Error(v ...interface{}) {
	GroupError(l.group, v...)
}

func (l *Log) Fatal(v ...interface{}) {
	GroupFatal(l.group, v...)
}
