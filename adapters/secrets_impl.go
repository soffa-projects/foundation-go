package adapters

import (
	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

func NewSecretProvider(provider string) f.SecretsProvider {
	if provider == "" {
		return nil
	}
	cfg, err := h.ParseUrl(provider)
	if err != nil {
		log.Fatal("failed to parse secret provider: %v", err)
	}
	switch cfg.Scheme {
	case "vault+https", "vault+http":
		return NewVaultSecretProvider(cfg)
	case "faker":
		return NewFakeSecretProvider()
	default:
		log.Fatal("unsupported secret provider: %s", cfg.Scheme)
	}
	return nil
}
