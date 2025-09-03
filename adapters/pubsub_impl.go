package adapters

import (
	"context"
	"errors"

	"github.com/go-redis/redis/v8"
	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

func NewPubSubProvider(provider string) f.PubSubProvider {
	res, err := h.ParseUrl(provider)
	if err != nil {
		log.Fatal("failed to parse pubsub provider: %v", err)
	}
	switch res.Scheme {
	case "redis":
		log.Info("using redis pubsub provider...")
		return NewRedisPubSubProvider(res)
	case "fake", "faker", "dummy":
		log.Info("using fake pubsub provider...")
		return NewFakePubSubProvider()
	default:
		log.Fatal("unsupported pubsub provider: %s", provider)
	}
	return nil
}

// ------------------------------------------------------------------------------------------------------------------
// REDIS PUBSUB PROVIDER IMPL
// ------------------------------------------------------------------------------------------------------------------

type RedisPubSubProvider struct {
	f.PubSubProvider
	client *redis.Client
}

func NewRedisPubSubProvider(cfg h.Url) f.PubSubProvider {
	db := 0
	if cfg.HasQueryParam("db") {
		db = int(cfg.Query("db").(int64))
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
	return &RedisPubSubProvider{
		client: client,
	}
}

func (p *RedisPubSubProvider) Init() error {
	return nil
}

func (p *RedisPubSubProvider) Publish(ctx context.Context, topic string, message string) error {
	err := p.client.Publish(ctx, topic, message).Err()
	if err != nil {
		log.Error("[redis]failed to publish message: %v", err)
		return err
	}
	log.Info("[redis]message published to topic: %s", topic)
	return nil
}

func (p *RedisPubSubProvider) Subscribe(ctx context.Context, topic string, handler func(message string)) {
	go func() {
		sub := p.client.Subscribe(ctx, topic)
		defer sub.Close()

		for {
			msg, err := sub.ReceiveMessage(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					// graceful shutdown
					return
				}
				log.Error("[redis] failed to receive message: %v", err)
				continue
			}
			log.Debug("[redis] event received: %s", msg.Payload)
			go handler(msg.Payload)
		}
	}()
}

func (p *RedisPubSubProvider) Ping() error {
	return p.client.Ping(context.Background()).Err()
}

// ------------------------------------------------------------------------------------------------------------------
// FAKE PUBSUB PROVIDER IMPL
// ------------------------------------------------------------------------------------------------------------------

type FakePubSubProvider struct {
	f.PubSubProvider
	sent        map[string]int
	received    map[string]int
	subscribers map[string][]func(message string)
}

func NewFakePubSubProvider() f.PubSubProvider {
	return &FakePubSubProvider{
		sent:        make(map[string]int),
		received:    make(map[string]int),
		subscribers: make(map[string][]func(message string)),
	}
}

func (p *FakePubSubProvider) Ping() error {
	return nil
}

func (p *FakePubSubProvider) Init() error {
	return nil
}

func (p *FakePubSubProvider) Publish(ctx context.Context, topic string, message string) error {
	p.sent[topic]++
	handlers := p.subscribers[topic]
	for _, handler := range handlers {
		p.received[topic]++
		go handler(message)
	}
	return nil
}

func (p *FakePubSubProvider) Subscribe(ctx context.Context, topic string, handler func(message string)) {
	p.subscribers[topic] = append(p.subscribers[topic], handler)
}

func (p *FakePubSubProvider) Received(event string) int {
	return p.received[event]
}

func (p *FakePubSubProvider) Sent(event string) int {
	return p.sent[event]
}
