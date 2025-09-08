package adapters

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

type defaultJwtProvider struct {
	f.TokenProvider
	privkey   jwk.Key
	pubkey    jwk.Key
	issuer    string
	secretKey string
}

func NewTokenProvider(cfg f.JwtConfig) f.TokenProvider {

	var privkey jwk.Key
	var pubkey jwk.Key

	if cfg.JwkPrivateBase64 != "" {
		privateKeyBytes, err := base64.StdEncoding.DecodeString(cfg.JwkPrivateBase64)
		if err != nil {
			log.Fatal("failed to decode private key: %s\n", err)
		}

		publicKeyBytes, err := base64.StdEncoding.DecodeString(cfg.JwkPublicBase64)
		if err != nil {
			log.Fatal("failed to decode public key: %s\n", err)
		}

		privateKeyPEM := stripPEMHeaders(string(privateKeyBytes))
		publicKeyPEM := stripPEMHeaders(string(publicKeyBytes))

		privkey, err = jwk.ParseKey([]byte(privateKeyPEM))
		if err != nil {
			log.Fatal("failed to parse JWK: %s\n", err)
		}
		pubkey, err = jwk.ParseKey([]byte(publicKeyPEM))
		if err != nil {
			log.Fatal("failed to get public key: %s\n", err)
		}
	}
	return &defaultJwtProvider{
		privkey:   privkey,
		pubkey:    pubkey,
		issuer:    cfg.Issuer,
		secretKey: cfg.SecretKey,
	}
}

func stripPEMHeaders(pemString string) string {
	lines := strings.Split(pemString, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and PEM headers/footers
		if line == "" ||
			strings.HasPrefix(line, "-----BEGIN") ||
			strings.HasPrefix(line, "-----END") {
			continue
		}
		cleanLines = append(cleanLines, line)
	}

	return strings.Join(cleanLines, "\n")
}

func (p *defaultJwtProvider) Create(cfg f.CreateJwtConfig) (string, error) {
	issuer := cfg.Issuer
	if issuer == "" {
		issuer = p.issuer
	}
	builder := jwt.NewBuilder().
		JwtID(h.NewId("")).
		Issuer(issuer).
		IssuedAt(time.Now()).
		Subject(cfg.Subject).
		Audience(cfg.Audience).
		Expiration(time.Now().Add(cfg.Ttl))

	for k, v := range cfg.Claims {
		builder.Claim(k, v)
	}

	tok, err := builder.Build()

	if err != nil {
		return "", err
	}
	var signed []byte
	if p.privkey != nil {

		signed, err = jwt.Sign(tok, jwt.WithKey(jwa.RS256(), p.privkey))
		if err != nil {
			return "", fmt.Errorf("failed to sign token: %s", err)
		}
	} else if p.secretKey != "" {
		signed, err = jwt.Sign(tok, jwt.WithKey(jwa.HS256(), []byte(p.secretKey)))
		if err != nil {
			return "", fmt.Errorf("failed to sign token: %s", err)
		}
	} else {
		return "", fmt.Errorf("no private key or secret key found")
	}

	return string(signed), nil
}

func (p *defaultJwtProvider) Verify(token string) (jwt.Token, error) {
	if token == "" {
		return nil, nil
	}
	var tok jwt.Token
	var err error
	if p.pubkey != nil {
		tok, err = jwt.Parse([]byte(token), jwt.WithKey(jwa.RS256(), p.pubkey))
	} else if p.secretKey != "" {
		tok, err = jwt.Parse([]byte(token), jwt.WithKey(jwa.HS256(), []byte(p.secretKey)))
	} else {
		return nil, fmt.Errorf("no public key or secret key found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %s", err)
	}
	return tok, nil
}

func (p *defaultJwtProvider) VerifyWithIssuer(token string, issuer string) (jwt.Token, error) {
	var tok jwt.Token
	var err error
	if p.pubkey != nil {
		tok, err = jwt.Parse([]byte(token), jwt.WithKey(jwa.RS256(), p.pubkey))
	} else if p.secretKey != "" {
		tok, err = jwt.Parse([]byte(token), jwt.WithKey(jwa.HS256(), []byte(p.secretKey)))
	} else {
		return nil, fmt.Errorf("no public key or secret key found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %s", err)
	}
	iss, _ := tok.Issuer()
	if iss != issuer {
		return nil, fmt.Errorf("invalid issuer")
	}
	return tok, nil
}

type defaultCsrfTokenProvider struct {
	f.CsrfTokenProvider
	secret string
}

func NewCsrfTokenProvider() f.CsrfTokenProvider {
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
