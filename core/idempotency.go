package f

import (
	"context"
)

// IdempotencyStore provides storage for idempotency keys to prevent duplicate operations
// The TTL is configured at the store level, not per operation
type IdempotencyStore interface {
	// Get retrieves a job ID associated with an idempotency key
	// Returns empty string if key doesn't exist
	Get(ctx context.Context, key string) (string, error)

	// Set stores a job ID with an idempotency key using the store's configured TTL
	Set(ctx context.Context, key string, jobID string) error
}
