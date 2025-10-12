package f

import "context"

const SecretProviderKey = "secrets"

type SecretsProvider interface {
	Ping() error
	Init() error
	Close() error
	Get(ctx context.Context, path string) (map[string]any, error)
	Put(ctx context.Context, path string, value map[string]any) error
}
