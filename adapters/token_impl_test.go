package adapters

import (
	"testing"
	"time"

	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/test"
)

func init() {
	// Initialize ID generator for tests
	h.InitIdGenerator(0)
}

// ------------------------------------------------------------------------------------------------------------------
// JWT Token Provider Tests
// ------------------------------------------------------------------------------------------------------------------

func TestNewTokenProvider_WithJWK(t *testing.T) {
	assert := test.NewAssertions(t)

	publicKey, privateKey := test.JWKSBase64()
	cfg := f.JwtConfig{
		JwkPublicBase64:  publicKey,
		JwkPrivateBase64: privateKey,
		Issuer:           "test-issuer",
	}

	provider, err := NewTokenProvider(cfg)

	assert.Nil(err)
	assert.NotNil(provider)
	// Verify it implements the interface
	var _ f.TokenProvider = provider
}

func TestNewTokenProvider_WithSecretKey(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.JwtConfig{
		SecretKey: "test-secret-key-at-least-32-chars-long",
		Issuer:    "test-issuer",
	}

	provider, err := NewTokenProvider(cfg)

	assert.Nil(err)
	assert.NotNil(provider)
}

func TestNewTokenProvider_EmptyConfig(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.JwtConfig{}

	provider, err := NewTokenProvider(cfg)

	// Should succeed but won't be able to create/verify tokens
	assert.Nil(err)
	assert.NotNil(provider)
}

func TestNewTokenProvider_InvalidBase64(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.JwtConfig{
		JwkPublicBase64:  "invalid-base64!!!",
		JwkPrivateBase64: "invalid-base64!!!",
	}

	provider, err := NewTokenProvider(cfg)

	assert.NotNil(err)
	if provider != nil {
		t.Error("Expected nil provider for invalid base64")
	}
}

func TestMustNewTokenProvider_Success(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.JwtConfig{
		SecretKey: "test-secret-key-at-least-32-chars-long",
	}

	provider := MustNewTokenProvider(cfg)

	assert.NotNil(provider)
}

func TestMustNewTokenProvider_Panic(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.JwtConfig{
		JwkPublicBase64:  "invalid",
		JwkPrivateBase64: "invalid",
	}

	// Should panic
	defer func() {
		r := recover()
		assert.NotNil(r)
	}()

	MustNewTokenProvider(cfg)
	t.Error("Should have panicked")
}

// ------------------------------------------------------------------------------------------------------------------
// Create Token Tests
// ------------------------------------------------------------------------------------------------------------------

func TestTokenProvider_Create_WithJWK(t *testing.T) {
	assert := test.NewAssertions(t)

	publicKey, privateKey := test.JWKSBase64()
	cfg := f.JwtConfig{
		JwkPublicBase64:  publicKey,
		JwkPrivateBase64: privateKey,
		Issuer:           "test-issuer",
	}
	provider, _ := NewTokenProvider(cfg)

	// Create a token
	token, err := provider.Create(f.CreateJwtConfig{
		Subject:  "user123",
		Audience: []string{"app1"},
		Ttl:      1 * time.Hour,
	})

	assert.Nil(err)
	assert.NotEmpty(token)
}

func TestTokenProvider_Create_WithSecretKey(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.JwtConfig{
		SecretKey: "test-secret-key-at-least-32-chars-long",
		Issuer:    "test-issuer",
	}
	provider, _ := NewTokenProvider(cfg)

	// Create a token
	token, err := provider.Create(f.CreateJwtConfig{
		Subject:  "user123",
		Audience: []string{"app1"},
		Ttl:      1 * time.Hour,
	})

	assert.Nil(err)
	assert.NotEmpty(token)
}

func TestTokenProvider_Create_WithCustomClaims(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.JwtConfig{
		SecretKey: "test-secret-key-at-least-32-chars-long",
	}
	provider, _ := NewTokenProvider(cfg)

	// Create token with custom claims
	token, err := provider.Create(f.CreateJwtConfig{
		Subject: "user123",
		Ttl:     1 * time.Hour,
		Claims: map[string]any{
			"role":  "admin",
			"email": "user@example.com",
		},
	})

	assert.Nil(err)
	assert.NotEmpty(token)
}

func TestTokenProvider_Create_WithCustomIssuer(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.JwtConfig{
		SecretKey: "test-secret-key-at-least-32-chars-long",
		Issuer:    "default-issuer",
	}
	provider, _ := NewTokenProvider(cfg)

	// Create token with custom issuer
	token, err := provider.Create(f.CreateJwtConfig{
		Subject:  "user123",
		Issuer:   "custom-issuer",
		Audience: []string{"app1"},
		Ttl:      1 * time.Hour,
	})

	assert.Nil(err)
	assert.NotEmpty(token)
}

func TestTokenProvider_Create_WithoutKeyOrSecret(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.JwtConfig{
		// No key or secret
	}
	provider, _ := NewTokenProvider(cfg)

	// Should fail
	token, err := provider.Create(f.CreateJwtConfig{
		Subject: "user123",
		Ttl:     1 * time.Hour,
	})

	assert.NotNil(err)
	assert.Equals(token, "")
}

// ------------------------------------------------------------------------------------------------------------------
// Verify Token Tests
// ------------------------------------------------------------------------------------------------------------------

func TestTokenProvider_Verify_WithJWK(t *testing.T) {
	assert := test.NewAssertions(t)

	publicKey, privateKey := test.JWKSBase64()
	cfg := f.JwtConfig{
		JwkPublicBase64:  publicKey,
		JwkPrivateBase64: privateKey,
		Issuer:           "test-issuer",
	}
	provider, _ := NewTokenProvider(cfg)

	// Create a token
	tokenStr, _ := provider.Create(f.CreateJwtConfig{
		Subject:  "user123",
		Audience: []string{"app1"},
		Ttl:      1 * time.Hour,
	})

	// Verify the token
	token, err := provider.Verify(tokenStr)
	assert.Nil(err)
	assert.NotNil(token)

	// Check claims
	subject, _ := token.Subject()
	assert.Equals(subject, "user123")
}

func TestTokenProvider_Verify_WithSecretKey(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.JwtConfig{
		SecretKey: "test-secret-key-at-least-32-chars-long",
	}
	provider, _ := NewTokenProvider(cfg)

	// Create a token
	tokenStr, _ := provider.Create(f.CreateJwtConfig{
		Subject: "user456",
		Ttl:     1 * time.Hour,
	})

	// Verify the token
	token, err := provider.Verify(tokenStr)
	assert.Nil(err)
	assert.NotNil(token)

	// Check claims
	subject, _ := token.Subject()
	assert.Equals(subject, "user456")
}

func TestTokenProvider_Verify_InvalidToken(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.JwtConfig{
		SecretKey: "test-secret-key-at-least-32-chars-long",
	}
	provider, _ := NewTokenProvider(cfg)

	// Try to verify invalid token
	token, err := provider.Verify("invalid.token.here")
	assert.NotNil(err)
	if token != nil {
		t.Error("Expected nil token for invalid token string")
	}
}

func TestTokenProvider_Verify_EmptyToken(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.JwtConfig{
		SecretKey: "test-secret-key-at-least-32-chars-long",
	}
	provider, _ := NewTokenProvider(cfg)

	// Verify empty token - should return nil, nil
	token, err := provider.Verify("")
	assert.Nil(err)
	if token != nil {
		t.Error("Expected nil token for empty string")
	}
}

func TestTokenProvider_Verify_WithoutKeyOrSecret(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.JwtConfig{
		// No key or secret
	}
	provider, _ := NewTokenProvider(cfg)

	// Should fail
	token, err := provider.Verify("some.token.here")
	assert.NotNil(err)
	if token != nil {
		t.Error("Expected nil token when no key/secret configured")
	}
}

// ------------------------------------------------------------------------------------------------------------------
// Issuer Verification Tests
// ------------------------------------------------------------------------------------------------------------------

func TestTokenProvider_Verify_CheckIssuer(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.JwtConfig{
		SecretKey: "test-secret-key-at-least-32-chars-long",
		Issuer:    "test-issuer",
	}
	provider, _ := NewTokenProvider(cfg)

	// Create a token with issuer
	tokenStr, _ := provider.Create(f.CreateJwtConfig{
		Subject: "user123",
		Ttl:     1 * time.Hour,
	})

	// Verify and check issuer
	token, err := provider.Verify(tokenStr)
	assert.Nil(err)
	assert.NotNil(token)

	issuer, _ := token.Issuer()
	assert.Equals(issuer, "test-issuer")
}

func TestTokenProvider_Verify_CustomIssuer(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.JwtConfig{
		SecretKey: "test-secret-key-at-least-32-chars-long",
		Issuer:    "default-issuer",
	}
	provider, _ := NewTokenProvider(cfg)

	// Create a token with custom issuer
	tokenStr, _ := provider.Create(f.CreateJwtConfig{
		Subject: "user123",
		Issuer:  "custom-issuer",
		Ttl:     1 * time.Hour,
	})

	// Verify and check custom issuer
	token, err := provider.Verify(tokenStr)
	assert.Nil(err)

	issuer, _ := token.Issuer()
	assert.Equals(issuer, "custom-issuer")
}

// ------------------------------------------------------------------------------------------------------------------
// Custom Claims Tests
// ------------------------------------------------------------------------------------------------------------------

func TestTokenProvider_CustomClaims_RoundTrip(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.JwtConfig{
		SecretKey: "test-secret-key-at-least-32-chars-long",
	}
	provider, _ := NewTokenProvider(cfg)

	// Create token with custom claims
	tokenStr, _ := provider.Create(f.CreateJwtConfig{
		Subject: "user123",
		Ttl:     1 * time.Hour,
		Claims: map[string]any{
			"role":   "admin",
			"email":  "admin@example.com",
			"active": true,
		},
	})

	// Verify and check custom claims
	token, err := provider.Verify(tokenStr)
	assert.Nil(err)

	// Access custom claims using Get method
	var role string
	var email string
	var active bool
	token.Get("role", &role)
	token.Get("email", &email)
	token.Get("active", &active)

	assert.Equals(role, "admin")
	assert.Equals(email, "admin@example.com")
	assert.Equals(active, true)
}

// ------------------------------------------------------------------------------------------------------------------
// CSRF Token Provider Tests
// ------------------------------------------------------------------------------------------------------------------

func TestNewCsrfTokenProvider(t *testing.T) {
	assert := test.NewAssertions(t)

	provider := NewCsrfTokenProvider()

	assert.NotNil(provider)
	// Verify it implements the interface
	var _ f.CsrfTokenProvider = provider
}

func TestCsrfTokenProvider_Create(t *testing.T) {
	assert := test.NewAssertions(t)

	provider := NewCsrfTokenProvider()

	token, err := provider.Create(1 * time.Hour)

	assert.Nil(err)
	assert.NotEmpty(token)
}

func TestCsrfTokenProvider_Verify_Valid(t *testing.T) {
	assert := test.NewAssertions(t)

	provider := NewCsrfTokenProvider()

	// Create token
	token, _ := provider.Create(1 * time.Hour)

	// Verify it
	err := provider.Verify(token)
	assert.Nil(err)
}

func TestCsrfTokenProvider_Verify_Invalid(t *testing.T) {
	assert := test.NewAssertions(t)

	provider := NewCsrfTokenProvider()

	// Try to verify invalid token
	err := provider.Verify("invalid-csrf-token")
	assert.NotNil(err)
}

func TestCsrfTokenProvider_Verify_ExpiredToken(t *testing.T) {
	assert := test.NewAssertions(t)

	provider := NewCsrfTokenProvider()

	// Create token with 2 second duration
	// Note: CSRF uses Unix seconds, not milliseconds
	token, _ := provider.Create(2 * time.Second)

	// Wait for it to expire
	time.Sleep(3 * time.Second)

	// Verify should fail
	err := provider.Verify(token)
	assert.NotNil(err)
}

func TestCsrfTokenProvider_DifferentProviders(t *testing.T) {
	assert := test.NewAssertions(t)

	provider1 := NewCsrfTokenProvider()
	provider2 := NewCsrfTokenProvider()

	// Create token with provider1
	token, _ := provider1.Create(1 * time.Hour)

	// Try to verify with provider2 (different secret)
	err := provider2.Verify(token)
	assert.NotNil(err) // Should fail - different secrets
}

// ------------------------------------------------------------------------------------------------------------------
// Integration Tests
// ------------------------------------------------------------------------------------------------------------------

func TestTokenProvider_EndToEnd_JWK(t *testing.T) {
	assert := test.NewAssertions(t)

	// Setup: Create provider with JWKs
	publicKey, privateKey := test.JWKSBase64()
	cfg := f.JwtConfig{
		JwkPublicBase64:  publicKey,
		JwkPrivateBase64: privateKey,
		Issuer:           "test-app",
	}
	provider, err := NewTokenProvider(cfg)
	assert.Nil(err)

	// Test: Create token
	tokenStr, err := provider.Create(f.CreateJwtConfig{
		Subject:  "user@example.com",
		Audience: []string{"web-app", "mobile-app"},
		Ttl:      2 * time.Hour,
		Claims: map[string]any{
			"role":   "admin",
			"tenant": "acme-corp",
		},
	})
	assert.Nil(err)
	assert.NotEmpty(tokenStr)

	// Test: Verify token
	token, err := provider.Verify(tokenStr)
	assert.Nil(err)
	assert.NotNil(token)

	// Test: Check standard claims
	subject, _ := token.Subject()
	assert.Equals(subject, "user@example.com")

	issuer, _ := token.Issuer()
	assert.Equals(issuer, "test-app")

	// Test: Check custom claims
	var role string
	var tenant string
	token.Get("role", &role)
	token.Get("tenant", &tenant)
	assert.Equals(role, "admin")
	assert.Equals(tenant, "acme-corp")
}

func TestTokenProvider_EndToEnd_SecretKey(t *testing.T) {
	assert := test.NewAssertions(t)

	// Setup: Create provider with secret key
	cfg := f.JwtConfig{
		SecretKey: "my-super-secret-key-for-testing-123",
		Issuer:    "api-server",
	}
	provider, err := NewTokenProvider(cfg)
	assert.Nil(err)

	// Create and verify token
	tokenStr, _ := provider.Create(f.CreateJwtConfig{
		Subject: "service-account",
		Ttl:     24 * time.Hour,
	})

	token, err := provider.Verify(tokenStr)
	assert.Nil(err)

	subject, _ := token.Subject()
	assert.Equals(subject, "service-account")
}

// NOTE: These tests focus on JWT creation and verification using:
// - JWK keys from test.JWKSBase64()
// - Secret key (HMAC)
// - CSRF tokens
//
// All tests use in-memory operations, no external dependencies.
