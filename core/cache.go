package adapters

import (
	"context"
	"time"
)

type CacheProvider interface {
	Init() error
	Close() error
	Ping() error
	Get(ctx context.Context, key string) (any, error)
	Set(ctx context.Context, key string, value any, duration time.Duration) error
}
