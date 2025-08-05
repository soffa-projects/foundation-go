package adapters

import (
	"context"
	"io/fs"

	core "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/log"
	"github.com/soffa-projects/foundation-go/utils"
	"github.com/soffa-projects/foundation-go/utils/dates"
)

type MultiTenantDataSource struct {
	core.DataSource
	DefaultDatabaseURL string
	migrationsFS       fs.FS
	master             core.Connection
	tenants            map[string]core.Connection
}

func NewMultiTenantDS(defaultDatabaseURL string, opts ...core.DSOpt) *MultiTenantDataSource {
	ds := &MultiTenantDataSource{
		DefaultDatabaseURL: defaultDatabaseURL,
		tenants:            make(map[string]core.Connection),
		//migrationsFS:       opts.MigrationsFS,
	}
	for _, opt := range opts {
		if opt.MigrationsFS != nil {
			ds.migrationsFS = opt.MigrationsFS
		}
	}
	if err := ds.init(); err != nil {
		log.Fatal("failed to initialize data source: %v", err)
	}
	return ds
}

func (ds *MultiTenantDataSource) init() error {
	cnx, err := ds.newConnection("default", ds.DefaultDatabaseURL)
	if err != nil {
		return err
	}
	ds.master = cnx
	var entities []core.TenantEntity
	if _, err := ds.master.Query(context.Background(), &entities, core.QueryOpts{}); err != nil {
		return err
	}
	for _, tenant := range entities {
		cnx, err := ds.newConnection(*tenant.ID, tenant.DatabaseUrl)
		if err != nil {
			return err
		}
		ds.tenants[*tenant.ID] = cnx
	}
	return nil
}

type TenantAlreadyExistsError struct {
	error
	Value string
}

func (ds *MultiTenantDataSource) CreateTenant(slug string, name string, databaseUrl string) (*core.TenantEntity, error) {

	found, err := ds.master.ExistsBy(context.Background(), &core.TenantEntity{}, "slug = ?", slug)
	if err != nil {
		return nil, err
	}
	if found {
		return nil, TenantAlreadyExistsError{Value: slug}
	}

	tenantId := utils.NewId("t_")

	if databaseUrl == "" {
		databaseUrl = utils.AppendParamToUrl(ds.master.DatabaseUrl(), "schema", tenantId)
	}

	tenant := &core.TenantEntity{
		ID:          &tenantId,
		Slug:        slug,
		Name:        name,
		DatabaseUrl: databaseUrl,
		Status:      utils.StrPtr("active"),
		CreatedAt:   dates.NowP(),
		UpdatedAt:   dates.NowP(),
	}
	cnx, err := ds.newConnection(tenantId, databaseUrl)
	if err != nil {
		return nil, err
	}
	ds.tenants[tenantId] = cnx
	if err := ds.master.Insert(context.Background(), tenant); err != nil {
		return nil, err
	}
	return tenant, nil
}

func (ds *MultiTenantDataSource) GetTenantList() ([]core.TenantEntity, error) {
	entities := []core.TenantEntity{}
	if _, err := ds.master.Query(context.Background(), &entities, core.QueryOpts{}); err != nil {
		return nil, err
	}
	return entities, nil
}

func (ds *MultiTenantDataSource) newConnection(id string, databaseUrl string) (core.Connection, error) {
	cnx := connectionImpl{
		Id:      id,
		Url:     databaseUrl,
		Default: id == "default" || id == "master" || id == "public",
	}
	if ds.migrationsFS != nil {
		err := cnx.configure(ds.migrationsFS)
		if err != nil {
			return nil, err
		}
	}
	cnx.initialized = true
	return cnx, nil
}
