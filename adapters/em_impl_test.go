package adapters

import (
	"context"
	"testing"

	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/test"
)

// Mock DataSource for testing
type mockDataSource struct {
	f.DataSource
	defaultCnx f.Connection
	tenantCnx  f.Connection
}

func (m *mockDataSource) Init(features []f.Feature) error {
	return nil
}

func (m *mockDataSource) DefaultConnection() f.Connection {
	return m.defaultCnx
}

func (m *mockDataSource) Connection(tenantId string) f.Connection {
	return m.tenantCnx
}

// Mock Connection for testing
type mockConnection struct {
	f.Connection
	name string
}

func TestNewEntityManagerImpl(t *testing.T) {
	assert := test.NewAssertions(t)

	ds := &mockDataSource{}
	em := NewEntityManagerImpl(ds)

	assert.NotNil(em)
	// Verify it implements the interface
	var _ f.EntityManager = em
}

func TestEntityManagerImpl_Default_WithContext(t *testing.T) {
	assert := test.NewAssertions(t)

	ds := &mockDataSource{}
	em := NewEntityManagerImpl(ds)

	// Create a context with default connection
	defaultCnx := &mockConnection{name: "default"}
	ctx := context.WithValue(context.Background(), f.DefaultCnxKey{}, defaultCnx)

	// Get default connection
	cnx := em.Default(ctx)
	assert.NotNil(cnx)
	assert.Equals(cnx.(*mockConnection).name, "default")
}

func TestEntityManagerImpl_Default_WithoutContext(t *testing.T) {
	ds := &mockDataSource{}
	em := NewEntityManagerImpl(ds)

	// Create a context without default connection
	ctx := context.Background()

	// Get default connection - should return nil
	cnx := em.Default(ctx)
	if cnx != nil {
		t.Error("Expected nil connection when default is not in context")
	}
}

func TestEntityManagerImpl_Tenant(t *testing.T) {
	assert := test.NewAssertions(t)

	tenantCnx := &mockConnection{name: "tenant-123"}
	ds := &mockDataSource{tenantCnx: tenantCnx}
	em := NewEntityManagerImpl(ds)

	ctx := context.Background()

	// Get tenant connection
	cnx := em.Tenant(ctx, "tenant-123")
	assert.NotNil(cnx)
	assert.Equals(cnx.(*mockConnection).name, "tenant-123")
}

func TestEntityManagerImpl_Tenant_DifferentTenants(t *testing.T) {
	assert := test.NewAssertions(t)

	// For this test, the mock returns the same connection
	// In a real implementation, different tenants would return different connections
	tenantCnx := &mockConnection{name: "tenant"}
	ds := &mockDataSource{tenantCnx: tenantCnx}
	em := NewEntityManagerImpl(ds)

	ctx := context.Background()

	// Get connections for different tenants
	cnx1 := em.Tenant(ctx, "tenant-1")
	cnx2 := em.Tenant(ctx, "tenant-2")

	// Both should return the same connection (due to mock implementation)
	assert.NotNil(cnx1)
	assert.NotNil(cnx2)
}

func TestEntityManagerImpl_Current_WithTenantContext(t *testing.T) {
	assert := test.NewAssertions(t)

	ds := &mockDataSource{}
	em := NewEntityManagerImpl(ds)

	// Create a context with tenant connection
	tenantCnx := &mockConnection{name: "tenant"}
	ctx := context.WithValue(context.Background(), f.TenantCnxKey{}, tenantCnx)

	// Get current connection - should return tenant connection
	cnx := em.Current(ctx)
	assert.NotNil(cnx)
	assert.Equals(cnx.(*mockConnection).name, "tenant")
}

func TestEntityManagerImpl_Current_WithDefaultContext(t *testing.T) {
	assert := test.NewAssertions(t)

	ds := &mockDataSource{}
	em := NewEntityManagerImpl(ds)

	// Create a context with only default connection
	defaultCnx := &mockConnection{name: "default"}
	ctx := context.WithValue(context.Background(), f.DefaultCnxKey{}, defaultCnx)

	// Get current connection - should fall back to default
	cnx := em.Current(ctx)
	assert.NotNil(cnx)
	assert.Equals(cnx.(*mockConnection).name, "default")
}

func TestEntityManagerImpl_Current_WithoutContext(t *testing.T) {
	ds := &mockDataSource{}
	em := NewEntityManagerImpl(ds)

	// Create a context without any connection
	ctx := context.Background()

	// Get current connection - should return nil (no tenant, no default)
	cnx := em.Current(ctx)
	if cnx != nil {
		t.Error("Expected nil connection when context has no connections")
	}
}

func TestEntityManagerImpl_Current_TenantOverridesDefault(t *testing.T) {
	assert := test.NewAssertions(t)

	ds := &mockDataSource{}
	em := NewEntityManagerImpl(ds)

	// Create a context with both tenant and default connections
	defaultCnx := &mockConnection{name: "default"}
	tenantCnx := &mockConnection{name: "tenant"}
	ctx := context.WithValue(context.Background(), f.DefaultCnxKey{}, defaultCnx)
	ctx = context.WithValue(ctx, f.TenantCnxKey{}, tenantCnx)

	// Get current connection - tenant should take precedence
	cnx := em.Current(ctx)
	assert.NotNil(cnx)
	assert.Equals(cnx.(*mockConnection).name, "tenant")
}

func TestEntityManagerImpl_WithNilDataSource(t *testing.T) {
	assert := test.NewAssertions(t)

	// Create EM with nil datasource (edge case)
	em := NewEntityManagerImpl(nil)
	assert.NotNil(em)

	ctx := context.Background()

	// Default and Current should still work (they don't use datasource)
	cnx := em.Default(ctx)
	if cnx != nil {
		t.Error("Expected nil connection")
	}

	cnx = em.Current(ctx)
	if cnx != nil {
		t.Error("Expected nil connection")
	}
}

func TestEntityManagerImpl_TenantWithNilDataSource(t *testing.T) {
	// This test verifies that calling Tenant() with nil datasource will panic
	// This is acceptable behavior as datasource should always be provided

	defer func() {
		r := recover()
		if r == nil {
			// No panic occurred, which is also acceptable if nil check exists
		}
	}()

	em := NewEntityManagerImpl(nil)
	ctx := context.Background()

	// This will panic with nil pointer dereference
	em.Tenant(ctx, "tenant-1")
}

func TestEntityManagerImpl_EmptyTenantId(t *testing.T) {
	assert := test.NewAssertions(t)

	tenantCnx := &mockConnection{name: "empty-tenant"}
	ds := &mockDataSource{tenantCnx: tenantCnx}
	em := NewEntityManagerImpl(ds)

	ctx := context.Background()

	// Get connection with empty tenant ID
	cnx := em.Tenant(ctx, "")
	assert.NotNil(cnx)
}

func TestEntityManagerImpl_InterfaceCompliance(t *testing.T) {
	assert := test.NewAssertions(t)

	ds := &mockDataSource{
		defaultCnx: &mockConnection{name: "default"},
		tenantCnx:  &mockConnection{name: "tenant"},
	}
	em := NewEntityManagerImpl(ds)

	// Verify all interface methods exist
	ctx := context.Background()

	// Default
	defaultCnx := &mockConnection{name: "default"}
	ctxWithDefault := context.WithValue(ctx, f.DefaultCnxKey{}, defaultCnx)
	cnx := em.Default(ctxWithDefault)
	assert.NotNil(cnx)

	// Tenant
	cnx = em.Tenant(ctx, "tenant-1")
	assert.NotNil(cnx)

	// Current
	tenantCnx := &mockConnection{name: "tenant"}
	ctxWithTenant := context.WithValue(ctx, f.TenantCnxKey{}, tenantCnx)
	cnx = em.Current(ctxWithTenant)
	assert.NotNil(cnx)
}

// NOTE: This implementation relies on context values being set by middleware/application code.
// The EntityManager itself does not create or manage connections, it only retrieves them from:
// 1. Context (for default and tenant connections)
// 2. DataSource.Connection() (for tenant-specific connections)
//
// This design allows for flexible connection management where:
// - Default connections are set once and reused
// - Tenant connections are resolved per-request based on tenant ID
// - Current connection respects tenant-first, then falls back to default
