package micro

import "fmt"

type TenantLoader interface {
	GetTenantList(e *EntityManager) []TenantInfo
	Get(e *EntityManager, tenant string) (*TenantInfo, error)
}

type FixedTenantLoader struct {
	TenantLoader
	tenants []TenantInfo
}

type TenantInfo struct {
	Id  string
	Url string
}

// tenantIDKey is a custom type for context keys to avoid collisions
type TenantID struct{}
type DBIKey struct{}

func (f *FixedTenantLoader) GetTenantList(_ *EntityManager) []TenantInfo {
	return f.tenants
}

func (f *FixedTenantLoader) Get(_ *EntityManager, tenant string) (*TenantInfo, error) {
	for _, t := range f.tenants {
		if t.Id == tenant {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("tenant %s not found", tenant)
}

func NewFixedTenantLoader(tenants []TenantInfo) *FixedTenantLoader {
	return &FixedTenantLoader{tenants: tenants}
}
