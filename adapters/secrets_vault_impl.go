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

func (v VaultSecretProvider) Get(ctx context.Context, tenantId string, key string) (any, error) {

	secretPath := strings.ReplaceAll(v.path, "__tenant__", tenantId)
	val := h.DefaultCache().GetOrSet(fmt.Sprintf("vault:%s:%s", secretPath, key), func() (any, error) {
		s, err := v.client.Secrets.KvV2Read(ctx, secretPath, vault.WithMountPath(v.mount))
		if err != nil {
			return nil, fmt.Errorf("[vault] failed to retrieve secret: %s, %v", secretPath, err)
		}
		return s.Data.Data[key], nil
	})
	if val == nil {
		return nil, fmt.Errorf("[vault] failed to retrieve secret: %s", key)
	}
	return val, nil
}

func (v VaultSecretProvider) GetObject(ctx context.Context, tenantId string, key string) (map[string]any, error) {
	value, err := v.Get(ctx, tenantId, key)
	if err != nil {
		return nil, err
	}
	return value.(map[string]any), err
}
