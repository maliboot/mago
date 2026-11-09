package logx

func GroupTracef(group, format string, v ...interface{}) {
	logRusIns.WithField("log_group", group).Tracef(format, v...)
}

func GroupDebugf(group, format string, v ...interface{}) {
	logRusIns.WithField("log_group", group).Debugf(format, v...)
}

func GroupInfof(group, format string, v ...interface{}) {
	logRusIns.WithField("log_group", group).Infof(format, v...)
}

func GroupWarnf(group, format string, v ...interface{}) {
	logRusIns.WithField("log_group", group).Warnf(format, v...)
}

func GroupErrorf(group, format string, v ...interface{}) {
	logRusIns.WithField("log_group", group).Errorf(format, v...)
}

func GroupFatalf(group, format string, v ...interface{}) {
	logRusIns.WithField("log_group", group).Fatalf(format, v...)
}

func GroupTrace(group string, args ...interface{}) {
	logRusIns.WithField("log_group", group).Trace(args...)
}

func GroupDebug(group string, args ...interface{}) {
	logRusIns.WithField("log_group", group).Debug(args...)
}

func GroupInfo(group string, args ...interface{}) {
	logRusIns.WithField("log_group", group).Info(args...)
}

func GroupWarn(group string, args ...interface{}) {
	logRusIns.WithField("log_group", group).Warn(args...)
}

func GroupError(group string, args ...interface{}) {
	logRusIns.WithField("log_group", group).Error(args...)
}

func GroupFatal(group string, args ...interface{}) {
	logRusIns.WithField("log_group", group).Fatal(args...)
}
