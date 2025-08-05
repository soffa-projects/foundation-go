package micro

import (
	log "github.com/sirupsen/logrus"
)

// log.Debugf("downloading invoice %s", input.ID)

func LogDebug(format string, args ...any) {
	log.Debugf(format, args...)
}

func LogInfo(format string, args ...any) {
	log.Infof(format, args...)
}

func LogWarn(format string, args ...any) {
	log.Warnf(format, args...)
}

func LogError(format string, args ...any) {
	log.Errorf(format, args...)
}

func LogFatal(format string, args ...any) {
	log.Fatalf(format, args...)
}
