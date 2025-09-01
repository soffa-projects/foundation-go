package adapters

import (
	"context"
	"fmt"
	"io/fs"

	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/log"
)

type MultiTenantDataSource struct {
	f.DataSource
	migrationsFS   fs.FS
	tenants        map[string]f.Connection
	tenantProvider f.TenantProvider
	features       []f.Feature
}

type DefaultDataSource struct {
	f.DataSource
	DatabaseURL string
}

const _defaultTenantId = "default"

// ------------------------------------------------------------------------------------------------------------------
// DATA SOURCE IMPL
// ------------------------------------------------------------------------------------------------------------------

func NewMultiTenantDS() f.DataSource {
	ds := &MultiTenantDataSource{
		tenants: make(map[string]f.Connection),
	}
	return ds
}

func (ds *MultiTenantDataSource) Init(env f.ApplicationEnv, features []f.Feature) error {
	if env.TenantProvider == nil {
		return fmt.Errorf("TENANT_PROVIDER_REQUIRED")
	}
	ds.tenantProvider = env.TenantProvider
	defaultTenant := env.TenantProvider.Default()
	cnx, err := newConnection(_defaultTenantId, defaultTenant.DatabaseUrl, nil, nil)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize data source: %v", err))
	}
	ds.tenants[_defaultTenantId] = cnx
	if err := ds.init(); err != nil {
		log.Fatal("failed to initialize data source: %v", err)
	}
	log.Info("multi tenant data source initialized with %d tenants", len(ds.tenants))
	return nil
}

func (ds *MultiTenantDataSource) init() error {
	if ds.tenantProvider == nil {
		return fmt.Errorf("tenant provider is not set")
	}
	tenantList, err := ds.tenantProvider.GetTenantList(context.Background())
	if err != nil {
		return err
	}

	for _, tenant := range tenantList {
		tenantId := *tenant.ID
		tenantSlug := *tenant.Slug
		if _, ok := ds.tenants[tenantId]; !ok {
			cnx, err := newConnection(tenantId, tenant.DatabaseUrl, ds.migrationsFS, ds.features)
			if err != nil {
				return err
			}
			ds.tenants[tenantId] = cnx
			ds.tenants[tenantSlug] = cnx
		}
	}
	return nil
}

func (ds *MultiTenantDataSource) Connection(id string) f.Connection {
	if cnx, ok := ds.tenants[id]; ok {
		return cnx
	}
	log.Debug("tenant connexion %s not found, initializing...", id)
	if err := ds.init(); err != nil {
		panic(fmt.Sprintf("tenant connexion %s not found", id))
	}
	if cnx, ok := ds.tenants[id]; ok {
		return cnx
	}
	panic(fmt.Sprintf("tenant connexion %s not found", id))
}

func newConnection(id string, databaseUrl string, migrationsFS fs.FS, features []f.Feature) (f.Connection, error) {
	cnx := connectionImpl{
		Id:      id,
		Url:     databaseUrl,
		Default: id == _defaultTenantId,
	}
	err := cnx.configure(migrationsFS, features)
	if err != nil {
		return nil, err
	}
	cnx.initialized = true
	return cnx, nil
}
