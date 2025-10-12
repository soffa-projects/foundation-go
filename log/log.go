package log

import (
	log "github.com/sirupsen/logrus"
)

// log.Debugf("downloading invoice %s", input.ID)

func Init(level string) {
	logLevel, err := log.ParseLevel(level)
	if err == nil {
		log.SetLevel(logLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
}

func IsDebugEnabled() bool {
	return log.GetLevel() == log.DebugLevel
}

func Debug(format string, args ...any) {
	log.Debugf(format, args...)
}

func Info(format string, args ...any) {
	log.Infof(format, args...)
}

func Warn(format string, args ...any) {
	log.Warnf(format, args...)
}

func Error(format string, args ...any) {
	log.Errorf(format, args...)
}

func Fatal(format string, args ...any) {
	log.Fatalf(format, args...)
}
