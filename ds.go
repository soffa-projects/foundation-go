package f

import "context"

type DataSource interface {
	CreateTenant(ctx context.Context, slug string, name string, databaseUrl string) (*TenantEntity, error)
	GetTenantList(ctx context.Context) ([]TenantEntity, error)
	GetTenant(ctx context.Context, id string) (*TenantEntity, error)
	TenantExists(ctx context.Context, id string) (bool, error)
	Connection(tenantId string) Connection
}
