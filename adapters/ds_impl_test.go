package adapters

import (
	"context"
	"testing"

	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/test"
)

// Mock TenantProvider for testing
type mockTenantProvider struct {
	f.TenantProvider
	tenants []f.Tenant
}

func (m *mockTenantProvider) Load(ctx context.Context) ([]f.Tenant, error) {
	return m.tenants, nil
}

// ------------------------------------------------------------------------------------------------------------------
// Constructor Tests
// ------------------------------------------------------------------------------------------------------------------

func TestNewMultiTenantDS(t *testing.T) {
	assert := test.NewAssertions(t)

	ds := NewMultiTenantDS()

	assert.NotNil(ds)
	// Verify it implements the interface
	var _ f.DataSource = ds
}

func TestNewMultiTenantDS_WithConfig(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.DataSourceConfig{
		DatabaseUrl: test.TestDatabaseURL(),
	}
	ds := NewMultiTenantDS(cfg)

	assert.NotNil(ds)
	assert.Equals(ds.cfg.DatabaseUrl, cfg.DatabaseUrl)
}

func TestNewMultiTenantDS_WithoutConfig(t *testing.T) {
	assert := test.NewAssertions(t)

	ds := NewMultiTenantDS()

	assert.NotNil(ds)
	// Config should be empty
	assert.Equals(ds.cfg.DatabaseUrl, "")
}

// ------------------------------------------------------------------------------------------------------------------
// UseTenantProvider Tests
// ------------------------------------------------------------------------------------------------------------------

func TestMultiTenantDS_UseTenantProvider(t *testing.T) {
	assert := test.NewAssertions(t)

	ds := NewMultiTenantDS()
	provider := &mockTenantProvider{}

	result := ds.UseTenantProvider(provider)

	assert.NotNil(result)
	assert.Equals(result, ds) // Should return self for chaining
}

// ------------------------------------------------------------------------------------------------------------------
// Init Tests
// ------------------------------------------------------------------------------------------------------------------

func TestMultiTenantDS_Init_WithDefaultConnection(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.DataSourceConfig{
		DatabaseUrl: test.TestDatabaseURL(),
	}
	ds := NewMultiTenantDS(cfg)

	err := ds.Init([]f.Feature{})
	assert.Nil(err)

	// Should have default connection
	defaultCnx := ds.DefaultConnection()
	assert.NotNil(defaultCnx)
}

func TestMultiTenantDS_Init_WithoutDatabaseUrl(t *testing.T) {
	assert := test.NewAssertions(t)

	ds := NewMultiTenantDS()

	err := ds.Init([]f.Feature{})
	assert.Nil(err)

	// Should not have default connection
	defaultCnx := ds.DefaultConnection()
	if defaultCnx != nil {
		t.Error("Expected nil default connection when no database URL provided")
	}
}

func TestMultiTenantDS_Init_WithTenantProvider(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.DataSourceConfig{
		DatabaseUrl: test.TestDatabaseURL(),
	}
	ds := NewMultiTenantDS(cfg)

	// Set up tenant provider with test tenants
	provider := &mockTenantProvider{
		tenants: []f.Tenant{
			{
				ID:          "tenant1",
				Slug:        "tenant-one",
				DatabaseUrl: test.TestDatabaseURL(),
			},
			{
				ID:          "tenant2",
				Slug:        "tenant-two",
				DatabaseUrl: test.TestDatabaseURL(),
			},
		},
	}
	ds.UseTenantProvider(provider)

	err := ds.Init([]f.Feature{})
	assert.Nil(err)

	// Should have connections for both tenants
	cnx1 := ds.Connection("tenant1")
	assert.NotNil(cnx1)

	cnx2 := ds.Connection("tenant2")
	assert.NotNil(cnx2)
}

func TestMultiTenantDS_Init_WithoutTenantProvider(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.DataSourceConfig{
		DatabaseUrl: test.TestDatabaseURL(),
	}
	ds := NewMultiTenantDS(cfg)

	// Don't set tenant provider
	err := ds.Init([]f.Feature{})
	assert.Nil(err) // Should not error, just log warning
}

// ------------------------------------------------------------------------------------------------------------------
// DefaultConnection Tests
// ------------------------------------------------------------------------------------------------------------------

func TestMultiTenantDS_DefaultConnection(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.DataSourceConfig{
		DatabaseUrl: test.TestDatabaseURL(),
	}
	ds := NewMultiTenantDS(cfg)
	ds.Init([]f.Feature{})

	cnx := ds.DefaultConnection()
	assert.NotNil(cnx)

	// Verify it's a working connection
	err := cnx.Ping()
	assert.Nil(err)
}

func TestMultiTenantDS_DefaultConnection_NotSet(t *testing.T) {
	ds := NewMultiTenantDS() // No database URL
	ds.Init([]f.Feature{})

	cnx := ds.DefaultConnection()
	if cnx != nil {
		t.Error("Expected nil default connection")
	}
}

// ------------------------------------------------------------------------------------------------------------------
// Connection Tests
// ------------------------------------------------------------------------------------------------------------------

func TestMultiTenantDS_Connection_ById(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.DataSourceConfig{
		DatabaseUrl: test.TestDatabaseURL(),
	}
	ds := NewMultiTenantDS(cfg)

	provider := &mockTenantProvider{
		tenants: []f.Tenant{
			{
				ID:          "tenant1",
				Slug:        "tenant-one",
				DatabaseUrl: test.TestDatabaseURL(),
			},
		},
	}
	ds.UseTenantProvider(provider)
	ds.Init([]f.Feature{})

	// Get connection by ID
	cnx := ds.Connection("tenant1")
	assert.NotNil(cnx)

	// Verify it works
	err := cnx.Ping()
	assert.Nil(err)
}

func TestMultiTenantDS_Connection_BySlug(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.DataSourceConfig{
		DatabaseUrl: test.TestDatabaseURL(),
	}
	ds := NewMultiTenantDS(cfg)

	provider := &mockTenantProvider{
		tenants: []f.Tenant{
			{
				ID:          "tenant1",
				Slug:        "tenant-one",
				DatabaseUrl: test.TestDatabaseURL(),
			},
		},
	}
	ds.UseTenantProvider(provider)
	ds.Init([]f.Feature{})

	// Get connection by slug
	cnx := ds.Connection("tenant-one")
	assert.NotNil(cnx)
}

func TestMultiTenantDS_Connection_SameForIdAndSlug(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.DataSourceConfig{
		DatabaseUrl: test.TestDatabaseURL(),
	}
	ds := NewMultiTenantDS(cfg)

	provider := &mockTenantProvider{
		tenants: []f.Tenant{
			{
				ID:          "tenant1",
				Slug:        "tenant-one",
				DatabaseUrl: test.TestDatabaseURL(),
			},
		},
	}
	ds.UseTenantProvider(provider)
	ds.Init([]f.Feature{})

	// Get connection by ID and slug
	cnxById := ds.Connection("tenant1")
	cnxBySlug := ds.Connection("tenant-one")

	// Should be the same connection
	assert.Equals(cnxById.DatabaseUrl(), cnxBySlug.DatabaseUrl())
}

func TestMultiTenantDS_Connection_NotFound(t *testing.T) {
	cfg := f.DataSourceConfig{
		DatabaseUrl: test.TestDatabaseURL(),
	}
	ds := NewMultiTenantDS(cfg)
	ds.Init([]f.Feature{})

	// Get non-existent connection
	cnx := ds.Connection("nonexistent-tenant")
	if cnx != nil {
		t.Error("Expected nil connection for non-existent tenant")
	}
}

// ------------------------------------------------------------------------------------------------------------------
// InitTenant Tests
// ------------------------------------------------------------------------------------------------------------------

func TestMultiTenantDS_InitTenant_NewTenant(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.DataSourceConfig{
		DatabaseUrl: test.TestDatabaseURL(),
	}
	ds := NewMultiTenantDS(cfg)
	ds.Init([]f.Feature{})

	// Initialize a new tenant
	tenant := f.Tenant{
		ID:          "new-tenant",
		Slug:        "new-tenant-slug",
		DatabaseUrl: test.TestDatabaseURL(),
	}
	err := ds.initTenant(tenant)
	assert.Nil(err)

	// Should be able to get connection
	cnx := ds.Connection("new-tenant")
	assert.NotNil(cnx)
}

func TestMultiTenantDS_InitTenant_DuplicateTenant(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.DataSourceConfig{
		DatabaseUrl: test.TestDatabaseURL(),
	}
	ds := NewMultiTenantDS(cfg)

	provider := &mockTenantProvider{
		tenants: []f.Tenant{
			{
				ID:          "tenant1",
				Slug:        "tenant-one",
				DatabaseUrl: test.TestDatabaseURL(),
			},
		},
	}
	ds.UseTenantProvider(provider)
	ds.Init([]f.Feature{})

	// Try to initialize the same tenant again
	tenant := f.Tenant{
		ID:          "tenant1",
		Slug:        "tenant-one",
		DatabaseUrl: test.TestDatabaseURL(),
	}
	err := ds.initTenant(tenant)
	assert.Nil(err) // Should not error, just skip
}

// ------------------------------------------------------------------------------------------------------------------
// Multiple Tenants Tests
// ------------------------------------------------------------------------------------------------------------------

func TestMultiTenantDS_MultipleTenants(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.DataSourceConfig{
		DatabaseUrl: test.TestDatabaseURL(),
	}
	ds := NewMultiTenantDS(cfg)

	provider := &mockTenantProvider{
		tenants: []f.Tenant{
			{ID: "tenant1", Slug: "t1", DatabaseUrl: test.TestDatabaseURL()},
			{ID: "tenant2", Slug: "t2", DatabaseUrl: test.TestDatabaseURL()},
			{ID: "tenant3", Slug: "t3", DatabaseUrl: test.TestDatabaseURL()},
		},
	}
	ds.UseTenantProvider(provider)
	ds.Init([]f.Feature{})

	// All tenants should be accessible
	assert.NotNil(ds.Connection("tenant1"))
	assert.NotNil(ds.Connection("tenant2"))
	assert.NotNil(ds.Connection("tenant3"))

	// By slug too
	assert.NotNil(ds.Connection("t1"))
	assert.NotNil(ds.Connection("t2"))
	assert.NotNil(ds.Connection("t3"))
}

// ------------------------------------------------------------------------------------------------------------------
// Edge Cases
// ------------------------------------------------------------------------------------------------------------------

func TestMultiTenantDS_EmptyTenantId(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.DataSourceConfig{
		DatabaseUrl: test.TestDatabaseURL(),
	}
	ds := NewMultiTenantDS(cfg)
	ds.Init([]f.Feature{})

	tenant := f.Tenant{
		ID:          "",
		Slug:        "empty-id",
		DatabaseUrl: test.TestDatabaseURL(),
	}
	err := ds.initTenant(tenant)
	assert.Nil(err)

	// Should be accessible by empty string
	cnx := ds.Connection("")
	assert.NotNil(cnx)
}

func TestMultiTenantDS_TenantWithoutDatabaseUrl(t *testing.T) {
	assert := test.NewAssertions(t)

	cfg := f.DataSourceConfig{
		DatabaseUrl: test.TestDatabaseURL(),
	}
	ds := NewMultiTenantDS(cfg)
	ds.Init([]f.Feature{})

	// Tenant without database URL should fail
	tenant := f.Tenant{
		ID:          "no-url-tenant",
		Slug:        "no-url",
		DatabaseUrl: "", // Empty
	}
	err := ds.initTenant(tenant)
	assert.NotNil(err) // Should error
}

func TestMultiTenantDS_Connect_InvalidUrl(t *testing.T) {
	ds := NewMultiTenantDS()

	_, err := ds.connect(f.ConnectionConfig{
		Id:          "test",
		DatabaseUrl: "invalid://url",
	})

	if err == nil {
		t.Error("Expected error for invalid database URL")
	}
}

// ------------------------------------------------------------------------------------------------------------------
// Integration Tests
// ------------------------------------------------------------------------------------------------------------------

func TestMultiTenantDS_EndToEnd(t *testing.T) {
	assert := test.NewAssertions(t)

	// Setup: Create data source with default connection
	cfg := f.DataSourceConfig{
		DatabaseUrl: test.TestDatabaseURL(),
	}
	ds := NewMultiTenantDS(cfg)

	// Setup: Add tenant provider
	provider := &mockTenantProvider{
		tenants: []f.Tenant{
			{ID: "acme", Slug: "acme-corp", DatabaseUrl: test.TestDatabaseURL()},
			{ID: "demo", Slug: "demo-org", DatabaseUrl: test.TestDatabaseURL()},
		},
	}
	ds.UseTenantProvider(provider)

	// Initialize
	err := ds.Init([]f.Feature{})
	assert.Nil(err)

	// Test: Default connection works
	defaultCnx := ds.DefaultConnection()
	assert.NotNil(defaultCnx)
	assert.Nil(defaultCnx.Ping())

	// Test: Tenant connections work
	acmeCnx := ds.Connection("acme")
	assert.NotNil(acmeCnx)
	assert.Nil(acmeCnx.Ping())

	demoCnx := ds.Connection("demo")
	assert.NotNil(demoCnx)
	assert.Nil(demoCnx.Ping())

	// Test: Can access by slug
	acmeBySlug := ds.Connection("acme-corp")
	assert.NotNil(acmeBySlug)
	assert.Equals(acmeBySlug.DatabaseUrl(), acmeCnx.DatabaseUrl())
}

// NOTE: These tests focus on in-memory SQLite databases for simplicity.
// PostgreSQL-specific features (schema strategy) are not tested here.
//
// Features tested:
// - Constructor with/without config
// - Tenant provider setup
// - Initialization with/without tenants
// - Default connection management
// - Tenant connection management (by ID and slug)
// - Multiple tenant handling
// - Edge cases (empty IDs, invalid URLs)
//
// Features NOT tested (require PostgreSQL):
// - Schema-based multi-tenancy strategy
// - Database migrations with real migration files
// - Event-based tenant creation (TenantCreatedEvent)
