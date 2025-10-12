package adapters

import (
	"context"
	"fmt"
	"time"

	f "github.com/soffa-projects/foundation-go/core"
)

// IdempotencyStore implements ports.IdempotencyStore using Redis
type IdempotencyStore struct {
	cache f.CacheProvider
	ttl   time.Duration // Default TTL for all idempotency keys
}

// NewIdempotencyStore creates a new Redis-backed idempotency store with a configured TTL
// The TTL determines how long idempotency keys are cached in Redis
func NewIdempotencyStore(cache f.CacheProvider, ttl time.Duration) *IdempotencyStore {
	return &IdempotencyStore{
		cache: cache,
		ttl:   ttl,
	}
}

// Get retrieves a job ID associated with an idempotency key
func (s *IdempotencyStore) Get(ctx context.Context, key string) (string, error) {
	result, err := s.cache.Get(ctx, s.formatKey(key))
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// Set stores a job ID with an idempotency key using the configured TTL
func (s *IdempotencyStore) Set(ctx context.Context, key string, jobID string) error {
	err := s.cache.Set(ctx, s.formatKey(key), jobID, s.ttl)
	if err != nil {
		return fmt.Errorf("failed to set idempotency key: %w", err)
	}

	return nil
}

// formatKey adds a prefix to idempotency keys to namespace them in Redis
func (s *IdempotencyStore) formatKey(key string) string {
	return fmt.Sprintf("idempotency:%s", key)
}
