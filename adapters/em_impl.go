package adapters

import (
	"context"

	f "github.com/soffa-projects/foundation-go/core"
)

type EntityManagerImpl struct {
	f.EntityManager
	ds f.DataSource
}

func NewEntityManagerImpl(ds f.DataSource) f.EntityManager {
	return &EntityManagerImpl{
		ds: ds,
	}
}

func (em *EntityManagerImpl) Default(ctx context.Context) f.Connection {
	defaultCnx := ctx.Value(f.DefaultCnxKey{})
	if defaultCnx == nil {
		return nil
	}
	return defaultCnx.(f.Connection)
}

func (em *EntityManagerImpl) Tenant(ctx context.Context, tenantId string) f.Connection {
	return em.ds.Connection(tenantId)
}

func (em *EntityManagerImpl) Current(ctx context.Context) f.Connection {
	tenantCnx := ctx.Value(f.TenantCnxKey{})
	if tenantCnx != nil {
		return tenantCnx.(f.Connection)
	}
	return em.Default(ctx)
}
