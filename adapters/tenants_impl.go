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

type TenantProviderConfig struct {
	DefaultDatabaseURL string
	MigrationsFS       fs.FS
}

func ParseTenantProvider(provider string, cfg TenantProviderConfig) f.TenantProvider {
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
// TENANT PROVIDER IMPL
// ------------------------------------------------------------------------------------------------------------------

type LocalTenantProvider struct {
	f.TenantProvider
	defaultDatabaseURL string
	migrationsFS       fs.FS
	ds                 f.Connection
	tenants            map[string]bool
}

func NewLocalTenantProvider(defaultDatabaseURL string, migrationsFS fs.FS) f.TenantProvider {
	ds := NewDefaultDataSource(defaultDatabaseURL, f.DSOpt{MigrationsFS: migrationsFS})
	if ds == nil {
		log.Fatal("failed to initialize data source.")
	}
	provider := &LocalTenantProvider{
		defaultDatabaseURL: defaultDatabaseURL,
		ds:                 ds,
		migrationsFS:       migrationsFS,
		tenants:            make(map[string]bool),
	}

	return provider
}

func (tp *LocalTenantProvider) Default() f.TenantEntity {
	return f.TenantEntity{
		DatabaseUrl: tp.defaultDatabaseURL,
	}
}

func (tp *LocalTenantProvider) GetTenantList(ctx context.Context) ([]f.TenantEntity, error) {
	entities := []f.TenantEntity{}
	if _, err := tp.ds.Query(ctx, &entities, f.QueryOpts{}); err != nil {
		return nil, err
	}
	return entities, nil
}

func (tp *LocalTenantProvider) GetTenant(ctx context.Context, id string) (*f.TenantEntity, error) {
	entity := f.TenantEntity{}
	value := strings.ToLower(id)
	empty, err := tp.ds.FindBy(ctx, &entity, "slug = ? OR id = ?", value, value)
	if err != nil {
		return nil, err
	}
	if empty {
		return nil, nil
	}
	return &entity, nil
}

func (tp *LocalTenantProvider) TenantExists(ctx context.Context, id string) (bool, error) {
	_, ok := tp.tenants[id]
	if !ok {
		return false, nil
	}
	return true, nil
}

func (tp *LocalTenantProvider) CreateTenant(ctx context.Context, slug string, name string, dbUrl string) (*f.TenantEntity, error) {

	found, err := tp.ds.ExistsBy(ctx, (*f.TenantEntity)(nil), "slug = ?", slug)
	if err != nil {
		return nil, err
	}
	if found {
		return nil, f.TenantAlreadyExistsError{Value: slug}
	}

	tenantId := h.NewId("t_")

	if dbUrl == "" {
		dbUrl = h.AppendParamToUrl(tp.ds.DatabaseUrl(), "schema", tenantId)
	}

	tenant := &f.TenantEntity{
		ID:          &tenantId,
		Slug:        h.TrimToNull(slug),
		Name:        h.TrimToNull(name),
		DatabaseUrl: dbUrl,
		Status:      h.StrPtr("active"),
		CreatedAt:   h.NowP(),
		UpdatedAt:   h.NowP(),
	}

	_, err = newConnection(tenantId, dbUrl, tp.migrationsFS)
	if err != nil {
		return nil, err
	}
	if err := tp.ds.Insert(ctx, tenant); err != nil {
		return nil, err
	}
	tp.tenants[slug] = true
	tp.tenants[tenantId] = true
	return tenant, nil
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

type FixedTenantProvider struct {
	f.TenantProvider
	tenants            map[string]f.TenantEntity
	master             f.Connection
	defaultDatabaseURL string
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

	tenants := make(map[string]f.TenantEntity)

	for _, tenant := range content.Tenants {
		tenants[tenant.ID] = f.TenantEntity{
			DatabaseUrl: tenant.DatabaseUrl,
			ID:          h.StrPtr(tenant.ID),
			Slug:        h.StrPtr(tenant.Slug),
			Name:        h.StrPtr(tenant.ID),
		}
		_, err := newConnection(tenant.ID, tenant.DatabaseUrl, migrationsFS)
		if err != nil {
			log.Fatal("failed to initialize data source for tenant %s: %v", tenant.ID, err)
		}

	}
	ds := NewDefaultDataSource(defaultDatabaseURL, f.DSOpt{MigrationsFS: migrationsFS})
	if ds == nil {
		log.Fatal("failed to initialize data source.")
	}

	log.Info("file tenant provider initialized with %d tenants", len(tenants))

	return &FixedTenantProvider{
		tenants:            tenants,
		defaultDatabaseURL: defaultDatabaseURL,
		master:             ds,
	}
}

func (tp *FixedTenantProvider) Default() f.TenantEntity {
	return f.TenantEntity{
		DatabaseUrl: tp.defaultDatabaseURL,
	}
}

func (tp *FixedTenantProvider) GetTenantList(ctx context.Context) ([]f.TenantEntity, error) {
	tenants := []f.TenantEntity{}
	for _, tenant := range tp.tenants {
		tenants = append(tenants, tenant)
	}
	return tenants, nil
}

func (tp *FixedTenantProvider) GetTenant(ctx context.Context, id string) (*f.TenantEntity, error) {
	if tenant, ok := tp.tenants[id]; ok {
		return &tenant, nil
	}
	return nil, nil
}

func (tp *FixedTenantProvider) TenantExists(ctx context.Context, id string) (bool, error) {
	if _, ok := tp.tenants[id]; ok {
		return true, nil
	}
	return false, nil
}
