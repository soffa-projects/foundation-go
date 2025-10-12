package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

func NewCacheProvider(provider string) (f.CacheProvider, error) {
	if provider == "memory" {
		return NewInMemoryCacheProvider(), nil
	}
	res, err := h.ParseUrl(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cache provider: %v", err)
	}
	switch res.Scheme {
	case "redis":
		log.Info("using redis cache provider...")
		return NewRedisCacheProvider(provider)
	default:
		return nil, fmt.Errorf("unsupported cache provider: %s", provider)
	}
}

func MustNewCacheProvider(provider string) f.CacheProvider {
	cache, err := NewCacheProvider(provider)
	if err != nil {
		panic(err)
	}
	return cache
}

// ------------------------------------------------------------------------------------------------------------------
// REDIS PUBSUB PROVIDER IMPL
// ------------------------------------------------------------------------------------------------------------------

type RedisCacheProvider struct {
	f.CacheProvider
	client *redis.Client
}

func NewRedisCacheProvider(url string) (f.CacheProvider, error) {
	client, err := NewRedisClient(url)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client: %v", err)
	}
	return &RedisCacheProvider{
		client: client,
	}, nil
}

func (p *RedisCacheProvider) Init() error {
	return nil
}

func (p *RedisCacheProvider) Close() error {
	return p.client.Close()
}

func (p *RedisCacheProvider) Set(ctx context.Context, key string, value any, duration time.Duration) error {
	return p.client.Set(ctx, key, value, duration).Err()
}

func (p *RedisCacheProvider) Get(ctx context.Context, key string) (any, error) {
	return p.client.Get(ctx, key).Result()
}

func (p *RedisCacheProvider) Ping() error {
	return p.client.Ping(context.Background()).Err()
}

// ------------------------------------------------------------------------------------------------------------------
// FAKE PUBSUB PROVIDER IMPL
// ------------------------------------------------------------------------------------------------------------------

type InMemoryCacheProvider struct {
	f.CacheProvider
	cache map[string]any
}

func NewInMemoryCacheProvider() f.CacheProvider {
	return InMemoryCacheProvider{
		cache: make(map[string]any),
	}
}

func (p InMemoryCacheProvider) Ping() error {
	return nil
}

func (p InMemoryCacheProvider) Init() error {
	return nil
}

func (p InMemoryCacheProvider) Close() error {
	return nil
}

func (p InMemoryCacheProvider) Set(ctx context.Context, key string, value any, duration time.Duration) error {
	p.cache[key] = value
	return nil
}

func (p InMemoryCacheProvider) Get(ctx context.Context, key string) (any, error) {
	return p.cache[key], nil
}
