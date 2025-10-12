package h

import (
	"errors"
	"testing"
	"time"

	"github.com/go-playground/assert/v2"
)

func TestNewCache(t *testing.T) {
	cache, err := NewCache()
	assert.Equal(t, err, nil)
	assert.NotEqual(t, cache, nil)
}

func TestDefaultCache(t *testing.T) {
	cache1 := DefaultCache()
	cache2 := DefaultCache()
	// Should return same instance
	assert.Equal(t, cache1, cache2)
}

func TestCache_SetAndGet(t *testing.T) {
	cache := MustNewCache()

	cache.Set("key1", "value1")
	cache.Set("key2", 123)
	cache.Set("key3", true)

	// Ristretto cache is async, need to wait for write to complete
	time.Sleep(10 * time.Millisecond)

	val1, ok1 := cache.Get("key1")
	assert.Equal(t, ok1, true)
	assert.Equal(t, val1, "value1")

	val2, ok2 := cache.Get("key2")
	assert.Equal(t, ok2, true)
	assert.Equal(t, val2, 123)

	val3, ok3 := cache.Get("key3")
	assert.Equal(t, ok3, true)
	assert.Equal(t, val3, true)
}

func TestCache_GetMissingKey(t *testing.T) {
	cache := MustNewCache()

	val, ok := cache.Get("nonexistent")
	assert.Equal(t, ok, false)
	assert.Equal(t, val, nil)
}

func TestCache_GetOrSet_CacheHit(t *testing.T) {
	cache := MustNewCache()
	cache.Set("existing", "cached-value")
	time.Sleep(10 * time.Millisecond) // Wait for async write

	callCount := 0
	result := cache.GetOrSet("existing", func() (any, error) {
		callCount++
		return "new-value", nil
	})

	assert.Equal(t, result, "cached-value")
	assert.Equal(t, callCount, 0) // Function should not be called
}

func TestCache_GetOrSet_CacheMiss(t *testing.T) {
	cache := MustNewCache()

	callCount := 0
	result := cache.GetOrSet("new-key", func() (any, error) {
		callCount++
		return "computed-value", nil
	})

	assert.Equal(t, result, "computed-value")
	assert.Equal(t, callCount, 1)

	time.Sleep(10 * time.Millisecond) // Wait for async write

	// Verify it's now cached
	val, ok := cache.Get("new-key")
	assert.Equal(t, ok, true)
	assert.Equal(t, val, "computed-value")
}

func TestCache_GetOrSet_WithError(t *testing.T) {
	cache := MustNewCache()

	result := cache.GetOrSet("error-key", func() (any, error) {
		return nil, errors.New("computation failed")
	})

	assert.Equal(t, result, nil)

	// Verify nothing was cached
	_, ok := cache.Get("error-key")
	assert.Equal(t, ok, false)
}
