package f

import (
	"time"

	"github.com/lestrrat-go/jwx/v3/jwt"
)

type JwtConfig struct {
	JwkPrivateBase64 string
	JwkPublicBase64  string
}

type CreateJwtConfig struct {
	Subject  string
	Issuer   string
	Audience []string
	Claims   map[string]any
	Ttl      time.Duration
}

type TokenProvider interface {
	Create(cfg CreateJwtConfig) (string, error)
	Verify(token string) (jwt.Token, error)
	VerifyWithIssuer(token string, issuer string) (jwt.Token, error)
}
