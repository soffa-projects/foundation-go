package adapters

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/hashicorp/vault-client-go"
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
	default:
		log.Fatal("unsupported secret provider: %s", cfg.Scheme)
	}
	return nil
}

type VaultSecretProvider struct {
	f.SecretsProvider
	path   string
	mount  string
	cache  *ristretto.Cache[string, string]
	client *vault.Client
}

func NewVaultSecretProvider(cfg h.Url) f.SecretsProvider {
	token := cfg.User
	p := cfg.Query("path")
	mount := cfg.Query("mount")
	if token == "" {
		log.Fatal("[vault] token is required")
	}
	if p == "" {
		log.Fatal("[vault] path is required")
		decodedValue, err := url.QueryUnescape(p.(string))
		if err != nil {
			log.Fatal("[vault] failed to decode path: %v", err)
		}
		p = decodedValue

	}
	if mount == "" {
		log.Fatal("[vault] mount is required")
	}
	cache, err := ristretto.NewCache(&ristretto.Config[string, string]{
		NumCounters: 100,    // number of keys to track frequency of (10M).
		MaxCost:     1 << 5, // maximum cost of cache (32MB).
		BufferItems: 64,     // number of keys per Get buffer.
	})
	if err != nil {
		log.Fatal("[vault] failed to create cache: %v", err)
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
		log.Fatal("[vault] failed to create client: %v", err)
	} else {
		log.Info("[vault] secret provider installed")
	}
	return VaultSecretProvider{
		mount:  mount.(string),
		path:   p.(string),
		cache:  cache,
		client: client,
	}
}

func (v VaultSecretProvider) Init() error {
	return nil
}

func (v VaultSecretProvider) Close() error {
	v.cache.Close()
	return nil
}

func (v VaultSecretProvider) Get(ctx context.Context, tenantId string, key string) (string, error) {
	if val, ok := v.cache.Get(key); ok {
		return val, nil
	}

	secretPath := strings.ReplaceAll(v.path, "__tenant__", tenantId)
	s, err := v.client.Secrets.KvV2Read(ctx, secretPath, vault.WithMountPath(v.mount))
	if err != nil {
		return "", fmt.Errorf("[vault] failed to retrieve secret: %s, %v", secretPath, err)
	}
	if value, ok := s.Data.Data[key]; ok {
		v.cache.SetWithTTL(key, value.(string), 1, 1*time.Hour)
		return value.(string), nil
	}

	return "", nil
}
