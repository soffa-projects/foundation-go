package h

import (
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

func TestGetClaimValues_SingleClaim(t *testing.T) {
	tok, _ := jwt.NewBuilder().
		Claim("role", "admin").
		Build()

	values := GetClaimValues(tok, "role")
	assert.Equal(t, len(values), 1)
	assert.Equal(t, values[0], "admin")
}

func TestGetClaimValues_MultipleClaims(t *testing.T) {
	tok, _ := jwt.NewBuilder().
		Claim("role", "admin").
		Claim("permission", "write").
		Claim("grant", "read").
		Build()

	values := GetClaimValues(tok, "role", "permission", "grant")
	assert.Equal(t, len(values), 3)
	assert.Equal(t, values[0], "admin")
	assert.Equal(t, values[1], "write")
	assert.Equal(t, values[2], "read")
}

func TestGetClaimValues_MissingClaim(t *testing.T) {
	tok, _ := jwt.NewBuilder().
		Claim("role", "admin").
		Build()

	values := GetClaimValues(tok, "nonexistent")
	assert.Equal(t, len(values), 0)
}

func TestGetClaimValues_MixedExistingAndMissing(t *testing.T) {
	tok, _ := jwt.NewBuilder().
		Claim("role", "admin").
		Build()

	values := GetClaimValues(tok, "role", "nonexistent", "also-missing")
	assert.Equal(t, len(values), 1)
	assert.Equal(t, values[0], "admin")
}

func TestGetClaimValues_EmptyKeys(t *testing.T) {
	tok, _ := jwt.NewBuilder().
		Claim("role", "admin").
		Build()

	values := GetClaimValues(tok)
	assert.Equal(t, len(values), 0)
}

func TestGetClaimValues_EmptyClaimValue(t *testing.T) {
	tok, _ := jwt.NewBuilder().
		Claim("role", "").
		Build()

	values := GetClaimValues(tok, "role")
	assert.Equal(t, len(values), 0) // Empty strings are not included
}

func TestGetClaimValues_Integration(t *testing.T) {
	InitIdGenerator(0) // Initialize ID generator

	// Create a real JWT with the NewJwt function and verify we can extract claims
	config := JwtConfig{
		Subject:   "user123",
		SecretKey: "test-secret",
		Issuer:    "test-issuer",
		Audience:  []string{"api"},
		Claims: map[string]any{
			"permissions": "read:users",
			"role":        "admin",
		},
		Ttl: time.Hour,
	}

	tokenString, err := NewJwt(config)
	assert.Equal(t, err, nil)

	// Parse the token
	tok, err := jwt.Parse([]byte(tokenString), jwt.WithKey(jwa.HS256(), []byte("test-secret")))
	assert.Equal(t, err, nil)

	// Extract claims
	values := GetClaimValues(tok, "permissions", "role")
	assert.Equal(t, len(values), 2)
	assert.Equal(t, values[0], "read:users")
	assert.Equal(t, values[1], "admin")
}
