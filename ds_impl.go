package f

import (
	"context"
	"io/fs"
	"strings"

	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

type MultiTenantDataSource struct {
	DataSource
	DefaultDatabaseURL string
	migrationsFS       fs.FS
	master             Connection
	tenants            map[string]Connection
}

const _defaultTenantId = "default"

func NewMultiTenantDS(defaultDatabaseURL string, opts ...DSOpt) *MultiTenantDataSource {
	ds := &MultiTenantDataSource{
		DefaultDatabaseURL: defaultDatabaseURL,
		tenants:            make(map[string]Connection),
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
	cnx, err := ds.newConnection(_defaultTenantId, ds.DefaultDatabaseURL)
	if err != nil {
		return err
	}
	ds.master = cnx
	ds.tenants[_defaultTenantId] = cnx

	tenantList, err := ds.GetTenantList(context.Background())
	if err != nil {
		return err
	}

	for _, tenant := range tenantList {
		cnx, err := ds.newConnection(*tenant.ID, tenant.DatabaseUrl)
		if err != nil {
			return err
		}
		ds.tenants[*tenant.ID] = cnx
		ds.tenants[*tenant.Slug] = cnx
	}
	return nil
}

type TenantAlreadyExistsError struct {
	error
	Value string
}

func (ds *MultiTenantDataSource) Connection(id string) Connection {
	if cnx, ok := ds.tenants[id]; ok {
		return cnx
	}
	return nil
}
func (ds *MultiTenantDataSource) CreateTenant(ctx context.Context, slug string, name string, dbUrl string) (*TenantEntity, error) {

	found, err := ds.master.ExistsBy(ctx, (*TenantEntity)(nil), "slug = ?", slug)
	if err != nil {
		return nil, err
	}
	if found {
		return nil, TenantAlreadyExistsError{Value: slug}
	}

	tenantId := h.NewId("t_")

	if dbUrl == "" {
		dbUrl = h.AppendParamToUrl(ds.master.DatabaseUrl(), "schema", tenantId)
	}

	tenant := &TenantEntity{
		ID:          &tenantId,
		Slug:        h.TrimToNull(slug),
		Name:        h.TrimToNull(name),
		DatabaseUrl: dbUrl,
		Status:      h.StrPtr("active"),
		CreatedAt:   h.NowP(),
		UpdatedAt:   h.NowP(),
	}
	cnx, err := ds.newConnection(tenantId, dbUrl)
	if err != nil {
		return nil, err
	}
	ds.tenants[slug] = cnx
	ds.tenants[tenantId] = cnx
	if err := ds.master.Insert(ctx, tenant); err != nil {
		return nil, err
	}
	return tenant, nil
}

func (ds *MultiTenantDataSource) GetTenantList(ctx context.Context) ([]TenantEntity, error) {
	entities := []TenantEntity{}
	if _, err := ds.master.Query(ctx, &entities, QueryOpts{}); err != nil {
		return nil, err
	}
	return entities, nil
}

func (ds *MultiTenantDataSource) GetTenant(ctx context.Context, id string) (*TenantEntity, error) {
	entity := TenantEntity{}
	value := strings.ToLower(id)
	empty, err := ds.master.FindBy(ctx, &entity, "slug = ? OR id = ?", value, value)
	if err != nil {
		return nil, err
	}
	if empty {
		return nil, nil
	}
	return &entity, nil
}

func (ds *MultiTenantDataSource) TenantExists(ctx context.Context, id string) (bool, error) {
	_, ok := ds.tenants[id]
	if !ok {
		return false, nil
	}
	return true, nil
}

func (ds *MultiTenantDataSource) newConnection(id string, databaseUrl string) (Connection, error) {
	cnx := connectionImpl{
		Id:      id,
		Url:     databaseUrl,
		Default: id == _defaultTenantId,
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
