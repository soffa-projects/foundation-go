package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/soffa-projects/foundation-go/h"
)

func NewRedisClient(url string) (*redis.Client, error) {
	cfg, err := h.ParseUrl(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}
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
	_, err = client.Ping(context.Background()).Result()
	return client, err
}
