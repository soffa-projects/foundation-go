package h

import (
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/soffa-projects/foundation-go/log"
)

var defaultCache Cache

func DefaultCache() Cache {
	if defaultCache != nil {
		return defaultCache
	}

	defaultCache = NewCache()
	return defaultCache
}

type Cache interface {
	Get(key string) (any, bool)
	Set(key string, value any)
}

type cacheImpl struct {
	Cache
	internal *ristretto.Cache[string, any]
}

func NewCache() Cache {
	internal, err := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: 1000,
		MaxCost:     1000,
		BufferItems: 64,
	})
	if err != nil {
		log.Fatal("failed to create cache: %v", err)
	}
	return &cacheImpl{
		internal: internal,
	}
}

func (c *cacheImpl) Get(key string) (any, bool) {
	return c.internal.Get(key)
}

func (c *cacheImpl) Set(key string, value any) {
	c.internal.SetWithTTL(key, value, 1, 1*time.Hour)
}
