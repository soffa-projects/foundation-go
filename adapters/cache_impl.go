package adapters

import (
	"context"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

func NewCacheProvider(provider string) f.CacheProvider {
	if provider == "memory" {
		return NewInMemoryCacheProvider()
	}
	res, err := h.ParseUrl(provider)
	if err != nil {
		log.Fatal("failed to parse cache provider: %v", err)
	}
	switch res.Scheme {
	case "redis":
		log.Info("using redis cache provider...")
		return NewRedisCacheProvider(res)
	default:
		log.Fatal("unsupported cache provider: %s", provider)
	}
	return nil
}

// ------------------------------------------------------------------------------------------------------------------
// REDIS PUBSUB PROVIDER IMPL
// ------------------------------------------------------------------------------------------------------------------

type RedisCacheProvider struct {
	f.CacheProvider
	client *redis.Client
}

func NewRedisCacheProvider(cfg h.Url) f.CacheProvider {
	db := 0
	if cfg.HasQueryParam("db") {
		db = int(cfg.Query("db").(int64))
	}
	if cfg.Path != "" {
		value := strings.TrimPrefix(cfg.Path, "/")
		db = h.ToInt(value)
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Host,
		Username: cfg.User,
		Password: cfg.Password,
		DB:       db,
	})
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal("failed to ping redis: %v", err)
	}
	log.Info("redis connection successful")
	return &RedisCacheProvider{
		client: client,
	}
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
