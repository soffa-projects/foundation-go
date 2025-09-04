package adapters

import (
	"fmt"

	"github.com/getsentry/sentry-go"
	f "github.com/soffa-projects/foundation-go/core"
)

type SentryErrorReporter struct {
	f.ErrorReporter
	client *sentry.Client
}

func NewSentryErrorReporter(dsn string, env string) f.ErrorReporter {
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         dsn,
		Environment: env,
	}); err != nil {
		fmt.Printf("Sentry initialization failed: %v\n", err)
	}
	return &SentryErrorReporter{
		client: sentry.CurrentHub().Client(),
	}
}
