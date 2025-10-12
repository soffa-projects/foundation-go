package h

import (
	"strings"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
)

func TestGenerateSecureRandomBytes(t *testing.T) {
	bytes1, err := GenerateSecureRandomBytes(32)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(bytes1), 32)

	bytes2, err := GenerateSecureRandomBytes(32)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(bytes2), 32)

	// Should generate different values
	assert.NotEqual(t, string(bytes1), string(bytes2))
}

func TestGenerateSecureRandomBytes_ZeroLength(t *testing.T) {
	bytes, err := GenerateSecureRandomBytes(0)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(bytes), 0)
}

func TestGenerateOAuth2Credentials_Defaults(t *testing.T) {
	clientID, clientSecret, err := GenerateOAuth2Credentials(0, 0)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(clientID), 32)
	assert.Equal(t, len(clientSecret) > 60, true) // Base64 encoded, no padding
}

func TestGenerateOAuth2Credentials_CustomLengths(t *testing.T) {
	clientID, clientSecret, err := GenerateOAuth2Credentials(16, 48)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(clientID), 16)
	assert.Equal(t, len(clientSecret) >= 48, true)
}

func TestGenerateOAuth2Credentials_Uniqueness(t *testing.T) {
	id1, secret1, err1 := GenerateOAuth2Credentials(32, 64)
	id2, secret2, err2 := GenerateOAuth2Credentials(32, 64)

	assert.Equal(t, err1, nil)
	assert.Equal(t, err2, nil)
	assert.NotEqual(t, id1, id2)
	assert.NotEqual(t, secret1, secret2)
}

func TestHashPassword(t *testing.T) {
	password := "mySecurePassword123!"
	hash, err := HashPassword(password)

	assert.Equal(t, err, nil)
	assert.NotEqual(t, hash, "")
	assert.NotEqual(t, hash, password)
	assert.Equal(t, strings.HasPrefix(hash, "$2a$"), true) // bcrypt hash prefix
}

func TestHashPassword_EmptyString(t *testing.T) {
	hash, err := HashPassword("")
	assert.Equal(t, err, nil)
	assert.NotEqual(t, hash, "")
}

func TestComparePassword_Valid(t *testing.T) {
	password := "myPassword123"
	hash, err := HashPassword(password)
	assert.Equal(t, err, nil)

	assert.Equal(t, ComparePassword(password, hash), true)
}

func TestComparePassword_Invalid(t *testing.T) {
	password := "myPassword123"
	hash, err := HashPassword(password)
	assert.Equal(t, err, nil)

	assert.Equal(t, ComparePassword("wrongPassword", hash), false)
}

func TestComparePassword_EmptyPassword(t *testing.T) {
	password := ""
	hash, err := HashPassword(password)
	assert.Equal(t, err, nil)

	assert.Equal(t, ComparePassword("", hash), true)
	assert.Equal(t, ComparePassword("notEmpty", hash), false)
}

func TestNewJwt(t *testing.T) {
	InitIdGenerator(0) // Initialize ID generator before test

	config := JwtConfig{
		Subject:   "user123",
		SecretKey: "my-secret-key",
		Issuer:    "test-app",
		Audience:  []string{"api"},
		Claims:    map[string]any{"role": "admin"},
		Ttl:       time.Hour,
	}

	token, err := NewJwt(config)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, token, "")
	assert.Equal(t, strings.Count(token, "."), 2) // JWT has 3 parts
}

func TestNewJwt_WithMultipleAudiences(t *testing.T) {
	InitIdGenerator(0)

	config := JwtConfig{
		Subject:   "user123",
		SecretKey: "my-secret-key",
		Issuer:    "test-app",
		Audience:  []string{"api", "web", "mobile"},
		Claims:    map[string]any{},
		Ttl:       time.Minute,
	}

	token, err := NewJwt(config)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, token, "")
}

func TestNewJwt_WithMultipleClaims(t *testing.T) {
	InitIdGenerator(0)

	config := JwtConfig{
		Subject:   "user123",
		SecretKey: "my-secret-key",
		Issuer:    "test-app",
		Audience:  []string{"api"},
		Claims: map[string]any{
			"role":     "admin",
			"tenantId": "tenant1",
			"email":    "user@example.com",
		},
		Ttl: time.Hour,
	}

	token, err := NewJwt(config)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, token, "")
}

func TestNewCsrf(t *testing.T) {
	secret := "csrf-secret-key"
	duration := time.Hour

	token, err := NewCsrf(secret, duration)
	assert.Equal(t, err, nil)
	assert.NotEqual(t, token, "")
	assert.Equal(t, strings.Contains(token, "."), true)
	assert.Equal(t, strings.Contains(token, ":"), true)
}

func TestNewCsrf_Uniqueness(t *testing.T) {
	secret := "csrf-secret-key"
	duration := time.Hour

	token1, _ := NewCsrf(secret, duration)
	token2, _ := NewCsrf(secret, duration)

	assert.NotEqual(t, token1, token2)
}

func TestVerifyCsrf_Valid(t *testing.T) {
	secret := "csrf-secret-key"
	duration := time.Hour

	token, err := NewCsrf(secret, duration)
	assert.Equal(t, err, nil)

	err = VerifyCsrf(secret, token)
	assert.Equal(t, err, nil)
}

func TestVerifyCsrf_InvalidFormat(t *testing.T) {
	secret := "csrf-secret-key"
	err := VerifyCsrf(secret, "invalid-token")
	assert.NotEqual(t, err, nil)
	assert.Equal(t, err.Error(), "invalid token format")
}

func TestVerifyCsrf_InvalidSignature(t *testing.T) {
	secret := "csrf-secret-key"
	duration := time.Hour

	token, _ := NewCsrf(secret, duration)

	// Verify with different secret
	err := VerifyCsrf("wrong-secret", token)
	assert.NotEqual(t, err, nil)
	assert.Equal(t, err.Error(), "invalid CSRF token signature")
}

func TestVerifyCsrf_Expired(t *testing.T) {
	secret := "csrf-secret-key"
	duration := -time.Hour // Already expired

	token, err := NewCsrf(secret, duration)
	assert.Equal(t, err, nil)

	err = VerifyCsrf(secret, token)
	assert.NotEqual(t, err, nil)
	assert.Equal(t, err.Error(), "CSRF token expired")
}

func TestVerifyCsrf_MalformedPayload(t *testing.T) {
	secret := "csrf-secret-key"

	// Token without colon in payload
	err := VerifyCsrf(secret, "payload.signature")
	assert.NotEqual(t, err, nil)
}
