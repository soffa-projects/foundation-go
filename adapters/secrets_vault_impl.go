package adapters

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/hashicorp/vault-client-go"
	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

type VaultSecretProvider struct {
	f.SecretsProvider
	mount  string
	cache  *ristretto.Cache[string, string]
	client *vault.Client
}

func NewVaultSecretProvider(cfg h.Url) (f.SecretsProvider, error) {
	token := cfg.User
	mount := cfg.Query("mount")
	if token == "" {
		return nil, fmt.Errorf("[vault] token is required")
	}
	if mount == "" {
		mount = "kv"
	}
	cache, err := ristretto.NewCache(&ristretto.Config[string, string]{
		NumCounters: 100,    // number of keys to track frequency of (10M).
		MaxCost:     1 << 5, // maximum cost of cache (32MB).
		BufferItems: 64,     // number of keys per Get buffer.
	})
	if err != nil {
		return nil, fmt.Errorf("[vault] failed to create cache: %v", err)
	}

	address := fmt.Sprintf("%s://%s", strings.TrimPrefix(cfg.Scheme, "vault+"), cfg.Host)
	client, err := vault.New(
		vault.WithAddress(address),
		vault.WithRequestTimeout(30*time.Second),
	)
	if err == nil {
		err = client.SetToken(token)
	}
	if err != nil {
		return nil, fmt.Errorf("[vault] failed to create client: %v", err)
	}
	log.Info("[vault] secret provider installed")
	return VaultSecretProvider{
		mount:  mount.(string),
		cache:  cache,
		client: client,
	}, nil
}

func MustNewVaultSecretProvider(cfg h.Url) f.SecretsProvider {
	provider, err := NewVaultSecretProvider(cfg)
	if err != nil {
		panic(err)
	}
	return provider
}

func (v VaultSecretProvider) Init() error {
	return nil
}

func (v VaultSecretProvider) Close() error {
	v.cache.Close()
	return nil
}

func (v VaultSecretProvider) Get(ctx context.Context, path string) (map[string]any, error) {
	val := h.DefaultCache().GetOrSet(fmt.Sprintf("vault:%s", path), func() (any, error) {
		s, err := v.client.Secrets.KvV2Read(ctx, path, vault.WithMountPath(v.mount))
		if err != nil {
			return nil, fmt.Errorf("[vault] failed to retrieve secret: %s, %v", path, err)
		}
		return s.Data.Data, nil
	})
	if val == nil {
		return nil, fmt.Errorf("[vault] failed to retrieve secret: %s", path)
	}
	return val.(map[string]any), nil
}

func (v VaultSecretProvider) Put(ctx context.Context, path string, value map[string]any) error {
	return fmt.Errorf("not implemented")
}
