package f

import "context"

const PubSubProviderKey = "pubsub"

type PubSubProvider interface {
	Init() error
	Publish(ctx context.Context, topic string, message any) error
	Subscribe(ctx context.Context, topic string, handler func(message any))
}
