package f

type DataSource interface {
	CreateTenant(slug string, name string, databaseUrl string) (*TenantEntity, error)
	GetTenantList() ([]TenantEntity, error)
	TenantExists(id string) (bool, error)
	Connection(tenantId string) Connection
}
