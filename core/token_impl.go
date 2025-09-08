package f

import (
	"time"

	"github.com/soffa-projects/foundation-go/h"
)

type defaultCsrfTokenProvider struct {
	CsrfTokenProvider
	secret string
}

func NewCsrfTokenProvider() CsrfTokenProvider {
	return &defaultCsrfTokenProvider{
		secret: h.RandomString(32),
	}
}

func (p *defaultCsrfTokenProvider) Create(duration time.Duration) (string, error) {
	return h.NewCsrf(p.secret, duration)
}

func (p *defaultCsrfTokenProvider) Verify(token string) error {
	return h.VerifyCsrf(p.secret, token)
}
