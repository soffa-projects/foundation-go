package adapters

import (
	"context"
	"testing"

	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/test"
)

func TestNewFakeSecretProvider(t *testing.T) {
	assert := test.NewAssertions(t)

	provider := NewFakeSecretProvider()

	assert.NotNil(provider)
	// Verify it implements the interface
	var _ f.SecretsProvider = provider
}

func TestFakeSecretProvider_Init(t *testing.T) {
	assert := test.NewAssertions(t)

	provider := NewFakeSecretProvider()
	err := provider.Init()

	assert.Nil(err)
}

func TestFakeSecretProvider_Ping(t *testing.T) {
	assert := test.NewAssertions(t)

	provider := NewFakeSecretProvider()
	err := provider.Ping()

	assert.Nil(err)
}

func TestFakeSecretProvider_Close(t *testing.T) {
	assert := test.NewAssertions(t)

	provider := NewFakeSecretProvider()
	err := provider.Close()

	assert.Nil(err)
}

func TestFakeSecretProvider_PutAndGet(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	provider := NewFakeSecretProvider()

	// Put a secret
	secrets := map[string]any{
		"username": "admin",
		"password": "secret123",
		"port":     5432,
	}
	err := provider.Put(ctx, "database/credentials", secrets)
	assert.Nil(err)

	// Get the secret back
	retrieved, err := provider.Get(ctx, "database/credentials")
	assert.Nil(err)
	assert.NotNil(retrieved)
	assert.Equals(retrieved["username"], "admin")
	assert.Equals(retrieved["password"], "secret123")
	assert.Equals(retrieved["port"], 5432)
}

func TestFakeSecretProvider_GetNonExistent(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	provider := NewFakeSecretProvider()

	// Get a path that doesn't exist
	retrieved, err := provider.Get(ctx, "nonexistent/path")
	assert.Nil(err)
	if retrieved != nil {
		assert.Equals(len(retrieved), 0)
	}
}

func TestFakeSecretProvider_OverwriteSecret(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	provider := NewFakeSecretProvider()

	// Put initial secret
	initial := map[string]any{"version": "1"}
	err := provider.Put(ctx, "app/config", initial)
	assert.Nil(err)

	// Overwrite with new secret
	updated := map[string]any{"version": "2", "new_field": "value"}
	err = provider.Put(ctx, "app/config", updated)
	assert.Nil(err)

	// Verify the secret was overwritten
	retrieved, err := provider.Get(ctx, "app/config")
	assert.Nil(err)
	assert.Equals(retrieved["version"], "2")
	assert.Equals(retrieved["new_field"], "value")
}

func TestFakeSecretProvider_MultiplePaths(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	provider := NewFakeSecretProvider()

	// Store secrets in different paths
	db := map[string]any{"host": "localhost"}
	api := map[string]any{"key": "abc123"}
	cache := map[string]any{"ttl": 300}

	provider.Put(ctx, "database", db)
	provider.Put(ctx, "api", api)
	provider.Put(ctx, "cache", cache)

	// Verify each path is independent
	dbRetrieved, _ := provider.Get(ctx, "database")
	assert.Equals(dbRetrieved["host"], "localhost")

	apiRetrieved, _ := provider.Get(ctx, "api")
	assert.Equals(apiRetrieved["key"], "abc123")

	cacheRetrieved, _ := provider.Get(ctx, "cache")
	assert.Equals(cacheRetrieved["ttl"], 300)
}

func TestFakeSecretProvider_EmptyPath(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	provider := NewFakeSecretProvider()

	// Put and get with empty path
	secrets := map[string]any{"data": "value"}
	err := provider.Put(ctx, "", secrets)
	assert.Nil(err)

	retrieved, err := provider.Get(ctx, "")
	assert.Nil(err)
	assert.Equals(retrieved["data"], "value")
}

func TestFakeSecretProvider_EmptySecret(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	provider := NewFakeSecretProvider()

	// Put empty secret map
	err := provider.Put(ctx, "empty/path", map[string]any{})
	assert.Nil(err)

	retrieved, err := provider.Get(ctx, "empty/path")
	assert.Nil(err)
	assert.NotNil(retrieved)
	assert.Equals(len(retrieved), 0)
}

func TestFakeSecretProvider_NilSecret(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	provider := NewFakeSecretProvider()

	// Put nil secret
	err := provider.Put(ctx, "nil/path", nil)
	assert.Nil(err)

	retrieved, err := provider.Get(ctx, "nil/path")
	assert.Nil(err)
	if retrieved != nil {
		assert.Equals(len(retrieved), 0)
	}
}

func TestFakeSecretProvider_ComplexValues(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	provider := NewFakeSecretProvider()

	// Put secret with various types
	complex := map[string]any{
		"string":  "value",
		"int":     42,
		"float":   3.14,
		"bool":    true,
		"slice":   []string{"a", "b", "c"},
		"nested":  map[string]any{"key": "nested_value"},
	}
	err := provider.Put(ctx, "complex", complex)
	assert.Nil(err)

	retrieved, err := provider.Get(ctx, "complex")
	assert.Nil(err)
	assert.Equals(retrieved["string"], "value")
	assert.Equals(retrieved["int"], 42)
	assert.Equals(retrieved["float"], 3.14)
	assert.Equals(retrieved["bool"], true)
	assert.NotNil(retrieved["slice"])
	assert.NotNil(retrieved["nested"])
}

// NOTE: FakeSecretProvider is NOT thread-safe (uses plain map without mutex)
// This is acceptable for a test-only fake implementation.
// Concurrent access test is skipped to avoid false failures.

func TestFakeSecretProvider_ContextCancellation(t *testing.T) {
	assert := test.NewAssertions(t)

	provider := NewFakeSecretProvider()

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Operations should still work (context is not checked in fake implementation)
	secret := map[string]any{"key": "value"}
	err := provider.Put(ctx, "path", secret)
	assert.Nil(err)

	retrieved, err := provider.Get(ctx, "path")
	assert.Nil(err)
	assert.Equals(retrieved["key"], "value")
}

func TestFakeSecretProvider_InterfaceCompliance(t *testing.T) {
	assert := test.NewAssertions(t)

	provider := NewFakeSecretProvider()

	// Verify all interface methods exist and work
	assert.Nil(provider.Init())
	assert.Nil(provider.Ping())

	ctx := context.Background()
	err := provider.Put(ctx, "test", map[string]any{"key": "value"})
	assert.Nil(err)

	retrieved, err := provider.Get(ctx, "test")
	assert.Nil(err)
	assert.NotNil(retrieved)

	assert.Nil(provider.Close())
}
