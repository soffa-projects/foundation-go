package adapters

import (
	"fmt"

	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
)

func NewSecretProvider(provider string) (f.SecretsProvider, error) {
	if provider == "" {
		return nil, nil
	}
	cfg, err := h.ParseUrl(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to parse secret provider: %v", err)
	}
	switch cfg.Scheme {
	case "vault+https", "vault+http":
		return NewVaultSecretProvider(cfg)
	case "faker":
		return NewFakeSecretProvider(), nil
	default:
		return nil, fmt.Errorf("unsupported secret provider: %s", cfg.Scheme)
	}
}

func MustNewSecretProvider(provider string) f.SecretsProvider {
	secret, err := NewSecretProvider(provider)
	if err != nil {
		panic(err)
	}
	return secret
}
