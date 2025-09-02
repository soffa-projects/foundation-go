package f

import "context"

type DataSource interface {
	Init(env ApplicationEnv, features []Feature) error
	Connection(tenantId string) Connection
}

type Tenant struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	DatabaseUrl string `json:"database_url"`
}

type TenantProvider interface {
	Init(features []Feature) error
	Default() Tenant
	RegisterTenant(ctx context.Context, tenant Tenant) error
	GetTenantList(ctx context.Context) ([]Tenant, error)
	GetTenant(ctx context.Context, id string) (*Tenant, error)
	TenantExists(ctx context.Context, id string) (bool, error)
}

type TenantAlreadyExistsError struct {
	error
	Value string
}
