package adapters

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/go-resty/resty/v2"
	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

func NewTenantProvider(provider string) f.TenantProvider {
	res, err := h.ParseUrl(provider)
	if err != nil {
		log.Fatal("failed to parse tenant provider: %v", err)
	}
	if res.Scheme == "file" {
		log.Info("using file tenant provider: %s", res.Url)
		return NewFileTenantProvider(res)
	}
	if res.Scheme == "https" || res.Scheme == "http" {
		log.Info("using http tenant provider: %s", res.Url)
		return NewHttpTenantProvider(res)
	}
	log.Fatal("unsupported tenant provider: %s", res.Scheme)
	return nil
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

func NewFileTenantProvider(cfg h.Url) f.TenantProvider {

	// Open the JSON file
	file, err := os.Open(strings.TrimPrefix(cfg.Url, "file://"))
	if err != nil {
		log.Fatal("Error opening file: %v", err)
	}
	defer file.Close()

	// Read file contents
	bytes, err := io.ReadAll(file)
	if err != nil {
		log.Fatal("Error reading file: %v", err)
	}

	// Unmarshal into struct
	var content TenantFile
	if err := json.Unmarshal(bytes, &content); err != nil {
		log.Fatal("Error parsing JSON: %v", err)
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
	}
}

func (tp *FileTenantProvider) Default() f.Tenant {
	return f.Tenant{}
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
	return HttpTenantProvider{
		bearer: cfg.User,
		target: cfg.Url,
		client: resty.New(),
	}
}

func (tp HttpTenantProvider) GetTenantList(ctx context.Context) ([]f.Tenant, error) {
	var tenants f.TenantList
	_, err := tp.client.R().
		SetResult(&tenants).
		SetAuthToken(tp.bearer).
		Get(tp.target)

	if err != nil {
		return nil, err
	}
	return tenants.Tenants, nil
}

func (tp HttpTenantProvider) GetTenant(ctx context.Context, id string) (*f.Tenant, error) {
	if tenant, ok := tp.tenants[id]; ok {
		return &tenant, nil
	}
	return nil, nil
}
