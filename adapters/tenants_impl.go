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
// TENANT PROVIDER IMPL
// ------------------------------------------------------------------------------------------------------------------

type LocalTenantProvider struct {
	f.TenantProvider
	defaultDatabaseURL string
	migrationsFS       fs.FS
	features           []f.Feature
	master             f.Connection
	tenants            map[string]bool
}

func NewLocalTenantProvider(defaultDatabaseURL string, migrationsFS fs.FS) f.TenantProvider {
	/*ds := NewDefaultDataSource(defaultDatabaseURL, migrationsFS)
	if ds == nil {
		log.Fatal("failed to initialize data source.")
	}*/
	provider := &LocalTenantProvider{
		defaultDatabaseURL: defaultDatabaseURL,
		//ds:                 ds,
		migrationsFS: migrationsFS,
		tenants:      make(map[string]bool),
	}

	return provider
}

func (tp *LocalTenantProvider) Init(features []f.Feature) error {
	cnx, err := newConnection(_defaultTenantId, tp.defaultDatabaseURL, tp.migrationsFS, features)
	if err != nil {
		return nil
	}
	tp.features = features
	tp.master = cnx
	tenants, err := tp.GetTenantList(context.Background())
	if err != nil {
		return err
	}
	for _, tenant := range tenants {

		_, err := newConnection(*tenant.ID, tp.defaultDatabaseURL, tp.migrationsFS, features)
		if err != nil {
			return nil
		}

		tp.tenants[*tenant.ID] = true
		tp.tenants[*tenant.Slug] = true
	}

	return nil
}

func (tp *LocalTenantProvider) Default() f.TenantEntity {
	return f.TenantEntity{
		DatabaseUrl: tp.defaultDatabaseURL,
	}
}

func (tp *LocalTenantProvider) GetTenantList(ctx context.Context) ([]f.TenantEntity, error) {
	entities := []f.TenantEntity{}
	if _, err := tp.master.Query(ctx, &entities, f.QueryOpts{}); err != nil {
		return nil, err
	}
	return entities, nil
}

func (tp *LocalTenantProvider) GetTenant(ctx context.Context, id string) (*f.TenantEntity, error) {
	entity := f.TenantEntity{}
	value := strings.ToLower(id)
	empty, err := tp.master.FindBy(ctx, &entity, "slug = ? OR id = ?", value, value)
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

func (tp *LocalTenantProvider) CreateTenant(ctx context.Context, model f.Tenant) (*f.TenantEntity, error) {

	found, err := tp.master.ExistsBy(ctx, (*f.TenantEntity)(nil), "slug = ?", model.Slug)
	if err != nil {
		return nil, err
	}
	if found {
		return nil, f.TenantAlreadyExistsError{Value: model.Slug}
	}

	tenantId := h.NewId("t_")
	dbUrl := model.DatabaseUrl

	if model.DatabaseUrl == "" {
		dbUrl = h.AppendParamToUrl(tp.master.DatabaseUrl(), "schema", tenantId)
	}

	tenant := &f.TenantEntity{
		ID:          &tenantId,
		Slug:        h.TrimToNull(model.Slug),
		Name:        h.TrimToNull(model.Name),
		ApiKey:      model.ApiKey,
		DatabaseUrl: dbUrl,
		Status:      h.StrPtr("active"),
		CreatedAt:   h.NowP(),
		UpdatedAt:   h.NowP(),
	}

	_, err = newConnection(tenantId, dbUrl, tp.migrationsFS, tp.features)
	if err != nil {
		return nil, err
	}
	if err := tp.master.Insert(ctx, tenant); err != nil {
		return nil, err
	}
	tp.tenants[model.Slug] = true
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

type FileTenantProvider struct {
	f.TenantProvider
	tenants            map[string]f.TenantEntity
	master             f.Connection
	defaultDatabaseURL string
	features           []f.Feature
	migrationsFS       fs.FS
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

	log.Info("file tenant provider initialized with %d tenants", len(tenants))
	for _, tenant := range content.Tenants {
		databaseUrl := strings.Replace(tenant.DatabaseUrl, "%RANDOM%", h.RandomString(5), 1)
		tenants[tenant.ID] = f.TenantEntity{
			DatabaseUrl: databaseUrl,
			ID:          h.StrPtr(tenant.ID),
			Slug:        h.StrPtr(tenant.Slug),
			Name:        h.StrPtr(tenant.ID),
		}
	}

	return &FileTenantProvider{
		tenants:            tenants,
		defaultDatabaseURL: defaultDatabaseURL,
		migrationsFS:       migrationsFS,
	}
}

func (tp *FileTenantProvider) Init(features []f.Feature) error {
	ds, err := newConnection(_defaultTenantId, tp.defaultDatabaseURL, tp.migrationsFS, features)
	if err != nil {
		log.Fatal("failed to initialize data source: %v", err)
	}
	if ds == nil {
		log.Fatal("failed to initialize data source.")
	}
	tp.master = ds

	tp.features = features
	for _, tenant := range tp.tenants {
		_, err := newConnection(*tenant.ID, tenant.DatabaseUrl, tp.migrationsFS, features)
		if err != nil {
			log.Fatal("failed to initialize data source for tenant %s: %v", *tenant.ID, err)
		}

	}
	return nil
}

func (tp *FileTenantProvider) Default() f.TenantEntity {
	return f.TenantEntity{
		DatabaseUrl: tp.defaultDatabaseURL,
	}
}

func (tp *FileTenantProvider) GetTenantList(ctx context.Context) ([]f.TenantEntity, error) {
	tenants := []f.TenantEntity{}
	for _, tenant := range tp.tenants {
		tenants = append(tenants, tenant)
	}
	return tenants, nil
}

func (tp *FileTenantProvider) GetTenant(ctx context.Context, id string) (*f.TenantEntity, error) {
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
