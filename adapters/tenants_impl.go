package adapters

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"strings"

	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

func NewTenantProvider(provider string, cfg f.TenantProviderConfig) f.TenantProvider {
	res, err := h.ParseUrl(provider)
	if err != nil {
		log.Fatal("failed to parse tenant provider: %v", err)
	}
	if res.Scheme == "file" {
		log.Info("using file tenant provider: %s", res.Url)
		return NewFileTenantProvider(res, cfg.DefaultDatabaseURL, cfg.MigrationsFS)
	} else {
		log.Fatal("unsupported tenant provider: %s", res.Scheme)
	}
	return nil
}

// ------------------------------------------------------------------------------------------------------------------
// FIXED TENANT PROVIDER IMPL
// ------------------------------------------------------------------------------------------------------------------

type tenantItem struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	DatabaseUrl string `json:"database_url"`
}

type TenantFile struct {
	Tenants []tenantItem `json:"tenants"`
}

type FileTenantProvider struct {
	f.TenantProvider
	tenants            map[string]f.Tenant
	master             f.Connection
	defaultDatabaseURL string
	migrationsFS       []fs.FS
}

func NewFileTenantProvider(cfg h.Url, defaultDatabaseURL string, migrationsFS fs.FS) f.TenantProvider {

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
		tenants:            tenants,
		defaultDatabaseURL: defaultDatabaseURL,
		migrationsFS:       []fs.FS{migrationsFS},
	}
}

func (tp *FileTenantProvider) Init(features []f.Feature) error {

	for _, feature := range features {
		if feature.FS != nil {
			tp.migrationsFS = append(tp.migrationsFS, feature.FS)
		}
	}
	ds, err := NewConnection(ConnectionConfig{
		Id:           _defaultTenantId,
		DatabaseUrl:  tp.defaultDatabaseURL,
		MigrationsFS: tp.migrationsFS,
	})
	if err != nil {
		log.Fatal("failed to initialize data source: %v", err)
	}
	if ds == nil {
		log.Fatal("failed to initialize data source.")
	}
	tp.master = ds

	for _, tenant := range tp.tenants {
		_, err := NewConnection(ConnectionConfig{
			Id:           tenant.ID,
			DatabaseUrl:  tenant.DatabaseUrl,
			MigrationsFS: tp.migrationsFS,
		})
		if err != nil {
			log.Fatal("failed to initialize data source for tenant %s: %v", tenant.ID, err)
		}

	}
	return nil
}

func (tp *FileTenantProvider) Default() f.Tenant {
	return f.Tenant{
		DatabaseUrl: tp.defaultDatabaseURL,
	}
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

func (tp *FileTenantProvider) TenantExists(ctx context.Context, id string) (bool, error) {
	if _, ok := tp.tenants[id]; ok {
		return true, nil
	}
	return false, nil
}
