package f

import "context"

type DataSource interface {
	Connection(tenantId string) Connection
}

type TenantProvider interface {
	Default() TenantEntity
	CreateTenant(ctx context.Context, slug string, name string, databaseUrl string) (*TenantEntity, error)
	GetTenantList(ctx context.Context) ([]TenantEntity, error)
	GetTenant(ctx context.Context, id string) (*TenantEntity, error)
	TenantExists(ctx context.Context, id string) (bool, error)
}

type TenantAlreadyExistsError struct {
	error
	Value string
}
