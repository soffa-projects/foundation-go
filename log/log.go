package log

import (
	log "github.com/sirupsen/logrus"
)

// log.Debugf("downloading invoice %s", input.ID)

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
