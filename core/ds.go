package f

import (
	"context"
	"io/fs"
)

type DataSource interface {
	Init(features []Feature) error
	DefaultConnection() Connection
	Connection(tenantId string) Connection
}

type Tenant struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name,omitempty"`
	AltID       string `json:"alt_id,omitempty"`
	DatabaseUrl string `json:"database_url,omitempty"`
}

type TenantList struct {
	Tenants []Tenant `json:"tenants"`
}

type TenantProvider interface {
	Load(ctx context.Context) ([]Tenant, error)
	GetTenantList(ctx context.Context) ([]Tenant, error)
	GetTenant(ctx context.Context, id string) (*Tenant, error)
}

type TenantAlreadyExistsError struct {
	error
	Value string
}

type DataSourceConfig struct {
	DatabaseUrl    string
	Prefix         string
	Strategy       string
	MigrationFS    fs.FS
	TenantProvider TenantProvider
}

type ConnectionConfig struct {
	Id          string
	DatabaseUrl string
}

type EntityManager interface {
	Default(ctx context.Context) Connection
	Tenant(ctx context.Context, tenantId string) Connection
	Current(ctx context.Context) Connection
}
