package f

import "context"

const SecretProviderKey = "secrets"

type SecretsProvider interface {
	Init() error
	Close() error
	Get(ctx context.Context, tenantId string, key string) (string, error)
}
