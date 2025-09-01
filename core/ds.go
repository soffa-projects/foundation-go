package f

import "context"

type DataSource interface {
	Init(env ApplicationEnv, features []Feature) error
	Connection(tenantId string) Connection
}

type Tenant struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	DatabaseUrl string `json:"database_url"`
	ApiKey      string `json:"api_key"`
}

type TenantProvider interface {
	Init(features []Feature) error
	Default() TenantEntity
	CreateTenant(ctx context.Context, input Tenant) (*TenantEntity, error)
	GetTenantList(ctx context.Context) ([]TenantEntity, error)
	GetTenant(ctx context.Context, id string) (*TenantEntity, error)
	TenantExists(ctx context.Context, id string) (bool, error)
}

type TenantAlreadyExistsError struct {
	error
	Value string
}
