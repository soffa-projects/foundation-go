package adapters

import (
	"context"
	"testing"
	"time"

	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/test"
)

// ------------------------------------------------------------------------------------------------------------------
// Factory Functions Tests
// ------------------------------------------------------------------------------------------------------------------

func TestNewCacheProvider_Memory(t *testing.T) {
	assert := test.NewAssertions(t)

	cache, err := NewCacheProvider("memory")

	assert.Nil(err)
	assert.NotNil(cache)
	// Verify it implements the interface
	var _ f.CacheProvider = cache
}

func TestNewCacheProvider_UnsupportedScheme(t *testing.T) {
	assert := test.NewAssertions(t)

	cache, err := NewCacheProvider("unsupported://localhost")

	assert.NotNil(err)
	if cache != nil {
		t.Error("Expected nil cache for unsupported scheme")
	}
}

func TestNewCacheProvider_InvalidURL(t *testing.T) {
	assert := test.NewAssertions(t)

	cache, err := NewCacheProvider(":::invalid:::")

	assert.NotNil(err)
	if cache != nil {
		t.Error("Expected nil cache for invalid URL")
	}
}

func TestMustNewCacheProvider_Success(t *testing.T) {
	assert := test.NewAssertions(t)

	// Should not panic with valid provider
	cache := MustNewCacheProvider("memory")

	assert.NotNil(cache)
}

func TestMustNewCacheProvider_Panic(t *testing.T) {
	assert := test.NewAssertions(t)

	// Should panic with invalid provider
	defer func() {
		r := recover()
		assert.NotNil(r)
	}()

	MustNewCacheProvider("invalid://provider")
	t.Error("Should have panicked")
}

// ------------------------------------------------------------------------------------------------------------------
// InMemoryCacheProvider Tests
// ------------------------------------------------------------------------------------------------------------------

func TestNewInMemoryCacheProvider(t *testing.T) {
	assert := test.NewAssertions(t)

	cache := NewInMemoryCacheProvider()

	assert.NotNil(cache)
	// Verify it implements the interface
	var _ f.CacheProvider = cache
}

func TestInMemoryCacheProvider_Init(t *testing.T) {
	assert := test.NewAssertions(t)

	cache := NewInMemoryCacheProvider()
	err := cache.Init()

	assert.Nil(err)
}

func TestInMemoryCacheProvider_Ping(t *testing.T) {
	assert := test.NewAssertions(t)

	cache := NewInMemoryCacheProvider()
	err := cache.Ping()

	assert.Nil(err)
}

func TestInMemoryCacheProvider_Close(t *testing.T) {
	assert := test.NewAssertions(t)

	cache := NewInMemoryCacheProvider()
	err := cache.Close()

	assert.Nil(err)
}

func TestInMemoryCacheProvider_SetAndGet(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cache := NewInMemoryCacheProvider()

	// Set a value
	err := cache.Set(ctx, "key1", "value1", 1*time.Minute)
	assert.Nil(err)

	// Get the value back
	value, err := cache.Get(ctx, "key1")
	assert.Nil(err)
	assert.Equals(value, "value1")
}

func TestInMemoryCacheProvider_GetNonExistent(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cache := NewInMemoryCacheProvider()

	// Get a key that doesn't exist
	value, err := cache.Get(ctx, "nonexistent")
	assert.Nil(err)
	if value != nil {
		t.Error("Expected nil value for nonexistent key")
	}
}

func TestInMemoryCacheProvider_OverwriteValue(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cache := NewInMemoryCacheProvider()

	// Set initial value
	cache.Set(ctx, "key", "value1", 1*time.Minute)

	// Overwrite with new value
	cache.Set(ctx, "key", "value2", 1*time.Minute)

	// Verify the value was overwritten
	value, err := cache.Get(ctx, "key")
	assert.Nil(err)
	assert.Equals(value, "value2")
}

func TestInMemoryCacheProvider_MultipleKeys(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cache := NewInMemoryCacheProvider()

	// Set multiple keys
	cache.Set(ctx, "key1", "value1", 1*time.Minute)
	cache.Set(ctx, "key2", 42, 1*time.Minute)
	cache.Set(ctx, "key3", true, 1*time.Minute)

	// Verify each key is independent
	v1, _ := cache.Get(ctx, "key1")
	assert.Equals(v1, "value1")

	v2, _ := cache.Get(ctx, "key2")
	assert.Equals(v2, 42)

	v3, _ := cache.Get(ctx, "key3")
	assert.Equals(v3, true)
}

func TestInMemoryCacheProvider_ComplexTypes(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cache := NewInMemoryCacheProvider()

	// Test with struct
	type Person struct {
		Name string
		Age  int
	}
	person := Person{Name: "John", Age: 30}
	cache.Set(ctx, "person", person, 1*time.Minute)

	retrieved, err := cache.Get(ctx, "person")
	assert.Nil(err)
	assert.Equals(retrieved, person)

	// Test with slice
	slice := []int{1, 2, 3}
	cache.Set(ctx, "slice", slice, 1*time.Minute)

	retrievedSlice, err := cache.Get(ctx, "slice")
	assert.Nil(err)
	assert.Equals(retrievedSlice, slice)

	// Test with map
	m := map[string]int{"a": 1, "b": 2}
	cache.Set(ctx, "map", m, 1*time.Minute)

	retrievedMap, err := cache.Get(ctx, "map")
	assert.Nil(err)
	assert.Equals(retrievedMap, m)
}

func TestInMemoryCacheProvider_EmptyKey(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cache := NewInMemoryCacheProvider()

	// Set and get with empty key
	err := cache.Set(ctx, "", "value", 1*time.Minute)
	assert.Nil(err)

	value, err := cache.Get(ctx, "")
	assert.Nil(err)
	assert.Equals(value, "value")
}

func TestInMemoryCacheProvider_NilValue(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cache := NewInMemoryCacheProvider()

	// Set nil value
	err := cache.Set(ctx, "nil-key", nil, 1*time.Minute)
	assert.Nil(err)

	value, err := cache.Get(ctx, "nil-key")
	assert.Nil(err)
	if value != nil {
		t.Error("Expected nil value")
	}
}

func TestInMemoryCacheProvider_ZeroDuration(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cache := NewInMemoryCacheProvider()

	// Set with zero duration (no expiration)
	err := cache.Set(ctx, "key", "value", 0)
	assert.Nil(err)

	// Should still be retrievable (in-memory doesn't implement TTL)
	value, err := cache.Get(ctx, "key")
	assert.Nil(err)
	assert.Equals(value, "value")
}

func TestInMemoryCacheProvider_NegativeDuration(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cache := NewInMemoryCacheProvider()

	// Set with negative duration
	err := cache.Set(ctx, "key", "value", -1*time.Second)
	assert.Nil(err)

	// Should still be retrievable (in-memory doesn't implement TTL)
	value, err := cache.Get(ctx, "key")
	assert.Nil(err)
	assert.Equals(value, "value")
}

func TestInMemoryCacheProvider_ContextCancellation(t *testing.T) {
	assert := test.NewAssertions(t)

	cache := NewInMemoryCacheProvider()

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Operations should still work (context is not checked in in-memory implementation)
	err := cache.Set(ctx, "key", "value", 1*time.Minute)
	assert.Nil(err)

	value, err := cache.Get(ctx, "key")
	assert.Nil(err)
	assert.Equals(value, "value")
}

func TestInMemoryCacheProvider_InterfaceCompliance(t *testing.T) {
	assert := test.NewAssertions(t)

	cache := NewInMemoryCacheProvider()

	// Verify all interface methods exist and work
	assert.Nil(cache.Init())
	assert.Nil(cache.Ping())

	ctx := context.Background()
	err := cache.Set(ctx, "test", "value", 1*time.Minute)
	assert.Nil(err)

	value, err := cache.Get(ctx, "test")
	assert.Nil(err)
	assert.NotNil(value)

	assert.Nil(cache.Close())
}

// NOTE: InMemoryCacheProvider does NOT implement TTL/expiration
// Values are stored indefinitely in memory until overwritten or the process restarts.
// This is acceptable for a simple in-memory cache for testing/development.

// NOTE: InMemoryCacheProvider is NOT thread-safe (uses plain map without mutex)
// Similar to FakeSecretProvider, this is acceptable for single-threaded test scenarios.
// For production use, Redis-backed cache should be used which is thread-safe.
