package f

import (
	"time"

	"github.com/lestrrat-go/jwx/v3/jwt"
)

type JwtConfig struct {
	Issuer           string
	SecretKey        string
	JwkPrivateBase64 string
	JwkPublicBase64  string
}

type CreateJwtConfig struct {
	Subject   string
	SecretKey string
	Issuer    string
	Audience  []string
	Claims    map[string]any
	Ttl       time.Duration
}

type TokenProvider interface {
	Create(cfg CreateJwtConfig) (string, error)
	Verify(token string) (jwt.Token, error)
}

type CsrfTokenProvider interface {
	Create(duration time.Duration) (string, error)
	Verify(token string) error
}
