package adapters

import (
	"context"

	f "github.com/soffa-projects/foundation-go/core"
)

func NewFakeSecretProvider() f.SecretsProvider {
	return &FakeSecretProvider{
		store: make(map[string](map[string]any)),
	}
}

type FakeSecretProvider struct {
	f.SecretsProvider
	store map[string](map[string]any)
}

func (p *FakeSecretProvider) Init() error {
	return nil
}

func (p *FakeSecretProvider) Ping() error {
	return nil
}

func (p *FakeSecretProvider) Close() error {
	return nil
}

func (p *FakeSecretProvider) Get(ctx context.Context, path string) (map[string]any, error) {
	return p.store[path], nil
}

func (p *FakeSecretProvider) Put(ctx context.Context, path string, value map[string]any) error {
	p.store[path] = value
	return nil
}
