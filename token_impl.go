package f

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/soffa-projects/foundation-go/log"
	"github.com/soffa-projects/foundation-go/utils"
)

type defaultJwtProvider struct {
	TokenProvider
	privkey jwk.Key
	pubkey  jwk.Key
}

func NewTokenProvider(cfg JwtConfig) TokenProvider {

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

	privkey, err := jwk.ParseKey([]byte(privateKeyPEM))
	if err != nil {
		log.Fatal("failed to parse JWK: %s\n", err)
	}
	pubkey, err := jwk.ParseKey([]byte(publicKeyPEM))
	if err != nil {
		log.Fatal("failed to get public key: %s\n", err)
	}
	return &defaultJwtProvider{
		privkey: privkey,
		pubkey:  pubkey,
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

func (p *defaultJwtProvider) Create(cfg CreateJwtConfig) (string, error) {
	builder := jwt.NewBuilder().
		JwtID(utils.NewId("")).
		Issuer(cfg.Issuer).
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

	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256(), p.privkey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %s", err)
	}

	return string(signed), nil
}

func (p *defaultJwtProvider) Verify(token string) (jwt.Token, error) {
	tok, err := jwt.Parse([]byte(token), jwt.WithKey(jwa.RS256(), p.pubkey))

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %s", err)
	}
	return tok, nil
}

func (p *defaultJwtProvider) VerifyWithIssuer(token string, issuer string) (jwt.Token, error) {
	tok, err := jwt.Parse([]byte(token), jwt.WithKey(jwa.RS256(), p.pubkey))

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %s", err)
	}
	iss, _ := tok.Issuer()
	if iss != issuer {
		return nil, fmt.Errorf("invalid issuer")
	}
	return tok, nil
}
