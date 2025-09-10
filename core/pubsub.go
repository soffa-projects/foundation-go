package f

import "context"

const PubSubProviderKey = "pubsub"

type PubSubProvider interface {
	Ping() error
	Init() error
	Publish(ctx context.Context, topic string, message string) error
	Subscribe(ctx context.Context, topic string, handler func(ctx context.Context, message string))
}
