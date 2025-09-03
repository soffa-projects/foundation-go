package f

import (
	"context"
	"io/fs"
)

type DataSource interface {
	Init(env ApplicationEnv, features []Feature) error
	DefaultConnection() Connection
	Connection(tenantId string) Connection
}

type Tenant struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	DatabaseUrl string `json:"database_url"`
}

type TenantList struct {
	Tenants []Tenant `json:"tenants"`
}

type TenantProvider interface {
	GetTenantList(ctx context.Context) ([]Tenant, error)
	GetTenant(ctx context.Context, id string) (*Tenant, error)
}

type TenantAlreadyExistsError struct {
	error
	Value string
}

type DataSourceConfig struct {
	DatabaseUrl string
	Prefix      string
	Strategy    string
	MigrationFS fs.FS
}

type ConnectionConfig struct {
	Id          string
	DatabaseUrl string
}
