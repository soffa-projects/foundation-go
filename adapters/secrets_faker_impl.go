package adapters

import (
	"context"

	f "github.com/soffa-projects/foundation-go/core"
)

func NewFakeSecretProvider() f.SecretsProvider {
	return &FakeSecretProvider{}
}

type FakeSecretProvider struct {
	f.SecretsProvider
}

func (p *FakeSecretProvider) Init() error {
	return nil
}

func (p *FakeSecretProvider) Close() error {
	return nil
}

func (p *FakeSecretProvider) Get(ctx context.Context, tenantId string, key string) (any, error) {
	return nil, nil
}

func (p *FakeSecretProvider) GetObject(ctx context.Context, tenantId string, key string) (map[string]any, error) {
	return nil, nil
}
