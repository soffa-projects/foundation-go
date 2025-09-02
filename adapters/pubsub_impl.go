package adapters

import (
	"context"

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
		return NewRedisPubSubProvider(res.Url)
	case "fake", "faker", "dummy":
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

func NewRedisPubSubProvider(url string) f.PubSubProvider {
	client := redis.NewClient(&redis.Options{
		Addr: url,
	})
	return &RedisPubSubProvider{
		client: client,
	}
}

func (p *RedisPubSubProvider) Init() error {
	return nil
}

func (p *RedisPubSubProvider) Publish(ctx context.Context, topic string, message any) error {
	err := p.client.Publish(ctx, topic, message).Err()
	if err != nil {
		log.Error("failed to publish message: %v", err)
		return err
	}
	log.Info("message published to topic: %s", topic)
	return nil
}

func (p *RedisPubSubProvider) Subscribe(ctx context.Context, topic string, handler func(message any)) {
	sub := p.client.Subscribe(ctx, topic)
	for {
		msg, err := sub.ReceiveMessage(ctx)
		if err != nil {
			log.Error("failed to receive message: %v", err)
			continue
		}
		go handler(msg.Payload)
	}
}

// ------------------------------------------------------------------------------------------------------------------
// FAKE PUBSUB PROVIDER IMPL
// ------------------------------------------------------------------------------------------------------------------

type FakePubSubProvider struct {
	f.PubSubProvider
	sent        map[string]int
	received    map[string]int
	subscribers map[string][]func(message any)
}

func NewFakePubSubProvider() f.PubSubProvider {
	return &FakePubSubProvider{
		sent:        make(map[string]int),
		received:    make(map[string]int),
		subscribers: make(map[string][]func(message any)),
	}
}

func (p *FakePubSubProvider) Init() error {
	return nil
}

func (p *FakePubSubProvider) Publish(ctx context.Context, topic string, message any) error {
	p.sent[topic]++
	handlers := p.subscribers[topic]
	for _, handler := range handlers {
		p.received[topic]++
		go handler(message)
	}
	return nil
}

func (p *FakePubSubProvider) Subscribe(ctx context.Context, topic string, handler func(message any)) {
	p.subscribers[topic] = append(p.subscribers[topic], handler)
}

func (p *FakePubSubProvider) Received(event string) int {
	return p.received[event]
}

func (p *FakePubSubProvider) Sent(event string) int {
	return p.sent[event]
}
