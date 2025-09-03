package adapters

import (
	"context"
	"fmt"
	"io/fs"

	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

type MultiTenantDataSource struct {
	f.DataSource
	migrationsFS   []fs.FS
	tenants        map[string]f.Connection
	tenantProvider f.TenantProvider
	cfg            f.DataSourceConfig
}

type DefaultDataSource struct {
	f.DataSource
}

const _defaultTenantId = "default"

// ------------------------------------------------------------------------------------------------------------------
// DATA SOURCE IMPL
// ------------------------------------------------------------------------------------------------------------------

func NewMultiTenantDS(cfg f.DataSourceConfig) f.DataSource {
	ds := &MultiTenantDataSource{
		tenants: make(map[string]f.Connection),
		cfg:     cfg,
	}
	migrationsFS := []fs.FS{}
	if ds.cfg.MigrationFS != nil {
		migrationsFS = append(migrationsFS, ds.cfg.MigrationFS)
	}
	ds.migrationsFS = migrationsFS
	return ds
}

func (ds *MultiTenantDataSource) Init(env f.ApplicationEnv, features []f.Feature) error {
	if env.TenantProvider == nil {
		return fmt.Errorf("TENANT_PROVIDER_REQUIRED")
	}
	ds.tenantProvider = env.TenantProvider

	for _, feature := range features {
		if feature.FS != nil {
			ds.migrationsFS = append(ds.migrationsFS, feature.FS)
		}
	}
	if ds.cfg.DatabaseUrl != "" {
		cnx, err := ds.connect(f.ConnectionConfig{
			Id:          _defaultTenantId,
			DatabaseUrl: ds.cfg.DatabaseUrl,
		})
		if err != nil {
			panic(fmt.Sprintf("failed to initialize data source: %v", err))
		}
		ds.tenants[_defaultTenantId] = cnx
	}
	ctx := context.Background()
	if err := ds.init(ctx); err != nil {
		log.Fatal("failed to initialize data source: %v", err)
	}
	log.Info("multi tenant data source initialized with %d tenants", len(ds.tenants))
	f.OnEvent(context.Background(), "tenant_created", func(data map[string]any) error {
		return ds.init(ctx)
	})
	return nil
}

func (ds *MultiTenantDataSource) init(ctx context.Context) error {
	if ds.tenantProvider == nil {
		return fmt.Errorf("tenant provider is not set")
	}
	tenantList, err := ds.tenantProvider.Load(ctx)
	if err != nil {
		return err
	}
	for _, tenant := range tenantList {
		tenantId := tenant.ID
		tenantSlug := tenant.Slug
		if _, ok := ds.tenants[tenantId]; !ok {

			dbUrl := tenant.DatabaseUrl
			if ds.cfg.Strategy == "schema" {
				dbUrl = h.AppendParamToUrl(ds.cfg.DatabaseUrl, "schema", tenantId)
			}
			cnx, err := ds.connect(f.ConnectionConfig{
				Id:          tenantId,
				DatabaseUrl: dbUrl,
			})
			log.Info("tenant %s (%s) connection initialized", tenantId, tenantSlug)
			if err != nil {
				return err
			}
			ds.tenants[tenantId] = cnx
			ds.tenants[tenantSlug] = cnx
		}
	}
	return nil
}

func (ds *MultiTenantDataSource) DefaultConnection() f.Connection {
	return ds.tenants[_defaultTenantId]
}

func (ds *MultiTenantDataSource) Connection(id string) f.Connection {
	if cnx, ok := ds.tenants[id]; ok {
		return cnx
	}
	log.Debug("tenant connexion %s not found, initializing...", id)
	if err := ds.init(context.Background()); err != nil {
		panic(fmt.Sprintf("tenant connexion %s not found", id))
	}
	if cnx, ok := ds.tenants[id]; ok {
		return cnx
	}
	return nil
	//panic(fmt.Sprintf("tenant connexion %s not found", id))
}

func (ds *MultiTenantDataSource) connect(config f.ConnectionConfig) (f.Connection, error) {
	cnx := connectionImpl{
		Id:      config.Id,
		Url:     config.DatabaseUrl,
		Default: config.Id == _defaultTenantId,
	}
	err := cnx.configure(ds.migrationsFS, ds.cfg.Prefix)
	if err != nil {
		return nil, err
	}
	cnx.initialized = true
	return cnx, nil
}
