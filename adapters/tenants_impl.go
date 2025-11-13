package adapters

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-resty/resty/v2"
	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

func NewTenantProvider(provider string) (f.TenantProvider, error) {
	if strings.HasPrefix(provider, "base64:") {
		log.Info("using base64 tenant provider: %s", provider)
		return NewBase64TenantProvider(strings.TrimPrefix(provider, "base64:")), nil
	}

	res, err := h.ParseUrl(provider)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tenant provider: %v", err)
	}
	if res.Scheme == "file" {
		log.Info("using file tenant provider: %s", res.Url)
		return NewFileTenantProvider(res)
	}
	if res.Scheme == "https" || res.Scheme == "http" {
		log.Info("using http tenant provider: %s", res.Url)
		return NewHttpTenantProvider(res), nil
	}
	return nil, fmt.Errorf("unsupported tenant provider: %s", res.Scheme)
}

func MustNewTenantProvider(provider string) f.TenantProvider {
	tp, err := NewTenantProvider(provider)
	if err != nil {
		panic(err)
	}
	return tp
}

// ------------------------------------------------------------------------------------------------------------------
// FIXED TENANT PROVIDER IMPL
// ------------------------------------------------------------------------------------------------------------------

type TenantFile struct {
	Tenants []f.Tenant `json:"tenants"`
}

type FileTenantProvider struct {
	f.TenantProvider
	tenants map[string]f.Tenant
}

func NewFileTenantProvider(cfg h.Url) (f.TenantProvider, error) {

	// Open the JSON file
	file, err := os.Open(strings.TrimPrefix(cfg.Url, "file://"))
	if err != nil {
		return nil, fmt.Errorf("Error opening file: %v", err)
	}
	defer file.Close()

	// Read file contents
	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("Error reading file: %v", err)
	}

	// Unmarshal into struct
	var content TenantFile
	if err := json.Unmarshal(bytes, &content); err != nil {
		return nil, fmt.Errorf("Error parsing JSON: %v", err)
	}

	tenants := make(map[string]f.Tenant)

	log.Info("file tenant provider initialized with %d tenants", len(tenants))
	for _, tenant := range content.Tenants {
		databaseUrl := strings.Replace(tenant.DatabaseUrl, "%RANDOM%", h.RandomString(5), 1)
		tenants[tenant.ID] = f.Tenant{
			DatabaseUrl: databaseUrl,
			ID:          tenant.ID,
			Slug:        tenant.Slug,
			Name:        tenant.ID,
		}
	}

	return &FileTenantProvider{
		tenants: tenants,
	}, nil
}

func (tp *FileTenantProvider) Load(ctx context.Context) ([]f.Tenant, error) {
	return tp.GetTenantList(ctx)
}

func (tp *FileTenantProvider) GetTenantList(ctx context.Context) ([]f.Tenant, error) {
	tenants := []f.Tenant{}
	for _, tenant := range tp.tenants {
		tenants = append(tenants, tenant)
	}
	return tenants, nil
}

func (tp *FileTenantProvider) GetTenant(ctx context.Context, id string) (*f.Tenant, error) {
	if tenant, ok := tp.tenants[id]; ok {
		return &tenant, nil
	}
	return nil, nil
}

// ------------------------------------------------------------------------------------------------------------------
// HTTP TENANT PROVIDER IMPL
// ------------------------------------------------------------------------------------------------------------------

type HttpTenantProvider struct {
	f.TenantProvider
	tenants map[string]f.Tenant
	target  string
	bearer  string
	client  *resty.Client
}

func NewHttpTenantProvider(cfg h.Url) f.TenantProvider {
	p := HttpTenantProvider{
		bearer:  cfg.User,
		target:  cfg.Url,
		client:  resty.New(),
		tenants: make(map[string]f.Tenant),
	}
	if _, err := p.Load(context.Background()); err != nil {
		log.Error("failed to load tenants: %v", err)
	}
	return p
}

func (tp HttpTenantProvider) Load(ctx context.Context) ([]f.Tenant, error) {
	defaultCache := h.DefaultCache()

	var tenants f.TenantList
	_, err := tp.client.R().
		SetResult(&tenants).
		SetAuthToken(tp.bearer).
		Get(tp.target)

	if err != nil {
		log.Error("failed to load tenants: %v", err)
		return nil, err
	}
	defaultCache.Set("tenants", tenants.Tenants)
	log.Info("[http-tenant] %d tenants loaded", len(tenants.Tenants))
	for _, tenant := range tenants.Tenants {
		tp.tenants[tenant.ID] = tenant
		tp.tenants[tenant.Slug] = tenant
		log.Info("[http-tenant] tenant registed %s/%s", tenant.ID, tenant.Slug)
	}
	return tenants.Tenants, nil
}

func (tp HttpTenantProvider) GetTenantList(ctx context.Context) ([]f.Tenant, error) {
	defaultCache := h.DefaultCache()
	if val, ok := defaultCache.Get("tenants"); ok {
		return val.([]f.Tenant), nil
	}
	return tp.Load(ctx)
}

func (tp HttpTenantProvider) GetTenant(ctx context.Context, id string) (*f.Tenant, error) {
	if tenant, ok := tp.tenants[id]; ok {
		return &tenant, nil
	}
	return nil, nil
}

// ------------------------------------------------------------------------------------------------------------------
// BASE64 TENANT PROVIDER IMPL
// ------------------------------------------------------------------------------------------------------------------

type Base64TenantProvider struct {
	f.TenantProvider
	tenants map[string]f.Tenant
}

func NewBase64TenantProvider(cfg string) f.TenantProvider {
	tenants := f.TenantList{}
	decoded, err := base64.StdEncoding.DecodeString(cfg)
	if err != nil {
		log.Fatal("failed to decode base64: %v", err)
	}
	if err := json.Unmarshal(decoded, &tenants); err != nil {
		log.Fatal("failed to unmarshal tenants: %v", err)
	}
	tenantsMap := make(map[string]f.Tenant)
	for _, tenant := range tenants.Tenants {
		tenantsMap[tenant.ID] = tenant
	}
	return &Base64TenantProvider{
		tenants: tenantsMap,
	}
}

func (tp *Base64TenantProvider) Load(ctx context.Context) ([]f.Tenant, error) {
	return tp.GetTenantList(ctx)
}

func (tp *Base64TenantProvider) GetTenantList(ctx context.Context) ([]f.Tenant, error) {
	tenants := []f.Tenant{}
	for _, tenant := range tp.tenants {
		tenants = append(tenants, tenant)
	}
	return tenants, nil
}

func (tp *Base64TenantProvider) GetTenant(ctx context.Context, id string) (*f.Tenant, error) {
	if tenant, ok := tp.tenants[id]; ok {
		return &tenant, nil
	}
	return nil, nil
}
