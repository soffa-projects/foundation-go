package h

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"golang.org/x/crypto/bcrypt"
)

func GenerateSecureRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return bytes, nil
}

func GenerateOAuth2Credentials(clientIDLength, clientSecretLength int) (string, string, error) {
	// Set defaults if zero values are passed
	if clientIDLength <= 0 {
		clientIDLength = 32
	}
	if clientSecretLength <= 0 {
		clientSecretLength = 64
	}

	// Generate Client ID (hex encoded)
	clientIDBytes := make([]byte, clientIDLength/2) // Hex encoding doubles the length
	_, err := rand.Read(clientIDBytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate client ID: %w", err)
	}
	clientID := hex.EncodeToString(clientIDBytes)

	// Generate Client Secret (base64 URL-safe encoded)
	secretBytesLength := clientSecretLength * 3 / 4 // Base64 encoding increases length by ~33%
	secretBytes := make([]byte, secretBytesLength)
	_, err = rand.Read(secretBytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate client secret: %w", err)
	}

	clientSecret := base64.URLEncoding.EncodeToString(secretBytes)
	clientSecret = strings.TrimRight(clientSecret, "=") // Remove padding

	return clientID, clientSecret, nil
}

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func ComparePassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

type JwtConfig struct {
	Subject   string
	SecretKey string
	Issuer    string
	Audience  []string
	Claims    map[string]any
	Ttl       time.Duration
}

func NewJwt(cfg JwtConfig) (string, error) {
	issuer := cfg.Issuer
	builder := jwt.NewBuilder().
		JwtID(NewId("")).
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
	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.HS256(), []byte(cfg.SecretKey)))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %s", err)
	}

	return string(signed), nil
}

func NewCsrf(secret string, duration time.Duration) (string, error) {
	// random 32 bytes
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(random)

	// add expiry (unix timestamp as string)
	exp := time.Now().Add(duration).Unix()
	expStr := strconv.FormatInt(exp, 10)
	payloadWithExp := payload + ":" + base64.RawURLEncoding.EncodeToString([]byte(expStr))

	// sign with HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payloadWithExp))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	// final token: payload:exp.signature
	return payloadWithExp + "." + signature, nil
}

func VerifyCsrf(secret string, token string) error {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return errors.New("invalid token format")
	}

	payloadWithExp := parts[0]
	signature := parts[1]

	// verify signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payloadWithExp))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return errors.New("invalid CSRF token signature")
	}

	// split payload:exp
	payloadParts := strings.Split(payloadWithExp, ":")
	if len(payloadParts) != 2 {
		return errors.New("invalid token payload")
	}

	expBytes, err := base64.RawURLEncoding.DecodeString(payloadParts[1])
	if err != nil {
		return errors.New("invalid expiry encoding")
	}

	exp, err := strconv.ParseInt(string(expBytes), 10, 64)
	if err != nil {
		return errors.New("invalid expiry value")
	}

	if time.Now().Unix() > exp {
		return errors.New("CSRF token expired")
	}

	return nil
}
