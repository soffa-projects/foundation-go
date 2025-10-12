package adapters

import (
	"context"
	"testing"

	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/test"
)

// Test entity for database operations
type TestUser struct {
	f.Entity `bun:"table:test_users"`
	ID       int64  `bun:",pk,autoincrement"`
	Name     string `bun:",notnull"`
	Email    string `bun:",unique"`
	Age      int
}

// ------------------------------------------------------------------------------------------------------------------
// Constructor Tests
// ------------------------------------------------------------------------------------------------------------------

func TestNewConnection_SQLite(t *testing.T) {
	assert := test.NewAssertions(t)

	cnx, err := NewConnection(test.TestDatabaseURL())

	assert.Nil(err)
	assert.NotNil(cnx)
	// Verify it implements the interface
	var _ f.Connection = cnx
}

func TestNewConnection_InvalidURL(t *testing.T) {
	assert := test.NewAssertions(t)

	cnx, err := NewConnection("invalid://database")

	assert.NotNil(err)
	if cnx != nil {
		t.Error("Expected nil connection for invalid URL")
	}
}

func TestNewConnection_EmptyURL(t *testing.T) {
	assert := test.NewAssertions(t)

	cnx, err := NewConnection("")

	assert.NotNil(err)
	if cnx != nil {
		t.Error("Expected nil connection for empty URL")
	}
}

// ------------------------------------------------------------------------------------------------------------------
// Ping Tests
// ------------------------------------------------------------------------------------------------------------------

func TestConnectionImpl_Ping(t *testing.T) {
	assert := test.NewAssertions(t)

	cnx, err := NewConnection(test.TestDatabaseURL())
	assert.Nil(err)

	// Ping should work
	err = cnx.Ping()
	assert.Nil(err)
}

// ------------------------------------------------------------------------------------------------------------------
// Transaction Tests
// ------------------------------------------------------------------------------------------------------------------

func TestConnectionImpl_Tx_BeginTransaction(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx, err := NewConnection(test.TestDatabaseURL())
	assert.Nil(err)

	// Begin transaction
	tx, err := cnx.Tx(ctx)
	assert.Nil(err)
	assert.NotNil(tx)
}

func TestConnectionImpl_Tx_Commit(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx, err := NewConnection(test.TestDatabaseURL())
	assert.Nil(err)

	// Begin transaction
	tx, err := cnx.Tx(ctx)
	assert.Nil(err)

	// Commit transaction
	err = tx.Commit()
	assert.Nil(err)
}

func TestConnectionImpl_Tx_Rollback(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx, err := NewConnection(test.TestDatabaseURL())
	assert.Nil(err)

	// Begin transaction
	tx, err := cnx.Tx(ctx)
	assert.Nil(err)

	// Rollback transaction
	err = tx.Rollback()
	assert.Nil(err)
}

func TestConnectionImpl_CommitWithoutTx(t *testing.T) {
	assert := test.NewAssertions(t)

	cnx, err := NewConnection(test.TestDatabaseURL())
	assert.Nil(err)

	// Commit without transaction should be no-op
	err = cnx.Commit()
	assert.Nil(err)
}

func TestConnectionImpl_RollbackWithoutTx(t *testing.T) {
	assert := test.NewAssertions(t)

	cnx, err := NewConnection(test.TestDatabaseURL())
	assert.Nil(err)

	// Rollback without transaction should be no-op
	err = cnx.Rollback()
	assert.Nil(err)
}

// ------------------------------------------------------------------------------------------------------------------
// Insert/Update/Delete Tests (Basic CRUD)
// ------------------------------------------------------------------------------------------------------------------

func setupTestTable(t *testing.T) f.Connection {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx, err := NewConnection(test.TestDatabaseURL())
	assert.Nil(err)

	// Create test table
	impl := cnx.(connectionImpl)
	_, err = impl.db.NewCreateTable().Model((*TestUser)(nil)).IfNotExists().Exec(ctx)
	assert.Nil(err)

	return cnx
}

func TestConnectionImpl_Insert(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Insert a user
	user := &TestUser{Name: "John Doe", Email: "john@example.com", Age: 30}
	err := cnx.Insert(ctx, user)
	assert.Nil(err)
	assert.True(user.ID > 0) // Auto-increment should set ID
}

func TestConnectionImpl_Update(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Insert a user
	user := &TestUser{Name: "John Doe", Email: "john@example.com", Age: 30}
	cnx.Insert(ctx, user)

	// Update the user
	user.Age = 31
	err := cnx.Update(ctx, user, "age")
	assert.Nil(err)

	// Verify update
	found := &TestUser{ID: user.ID}
	notFound, err := cnx.FindBy(ctx, found, "id = ?", user.ID)
	assert.Nil(err)
	assert.False(notFound)
	assert.Equals(found.Age, 31)
}

func TestConnectionImpl_Delete(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Insert a user
	user := &TestUser{Name: "John Doe", Email: "john@example.com", Age: 30}
	cnx.Insert(ctx, user)

	// Delete the user
	err := cnx.Delete(ctx, user)
	assert.Nil(err)

	// Verify deletion
	found := &TestUser{ID: user.ID}
	notFound, err := cnx.FindBy(ctx, found, "id = ?", user.ID)
	assert.Nil(err)
	assert.True(notFound) // Should not be found
}

// ------------------------------------------------------------------------------------------------------------------
// Query Tests
// ------------------------------------------------------------------------------------------------------------------

func TestConnectionImpl_FindBy_Found(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Insert a user
	user := &TestUser{Name: "John Doe", Email: "john@example.com", Age: 30}
	cnx.Insert(ctx, user)

	// Find by ID
	found := &TestUser{}
	notFound, err := cnx.FindBy(ctx, found, "id = ?", user.ID)
	assert.Nil(err)
	assert.False(notFound)
	assert.Equals(found.Name, "John Doe")
	assert.Equals(found.Email, "john@example.com")
}

func TestConnectionImpl_FindBy_NotFound(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Find non-existent user
	found := &TestUser{}
	notFound, err := cnx.FindBy(ctx, found, "id = ?", 99999)
	assert.Nil(err)
	assert.True(notFound)
}

func TestConnectionImpl_ExistsBy_Exists(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Insert a user
	user := &TestUser{Name: "John Doe", Email: "john@example.com", Age: 30}
	cnx.Insert(ctx, user)

	// Check if exists - need an instance, not nil
	entity := &TestUser{}
	exists, err := cnx.ExistsBy(ctx, entity, "email = ?", "john@example.com")
	assert.Nil(err)
	assert.True(exists)
}

func TestConnectionImpl_ExistsBy_NotExists(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Check if non-existent user exists - need an instance
	entity := &TestUser{}
	exists, err := cnx.ExistsBy(ctx, entity, "email = ?", "nonexistent@example.com")
	assert.Nil(err)
	assert.False(exists)
}

func TestConnectionImpl_Count(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Insert multiple users
	cnx.Insert(ctx, &TestUser{Name: "User1", Email: "user1@example.com", Age: 20})
	cnx.Insert(ctx, &TestUser{Name: "User2", Email: "user2@example.com", Age: 25})
	cnx.Insert(ctx, &TestUser{Name: "User3", Email: "user3@example.com", Age: 30})

	// Count all users
	count, err := cnx.Count(ctx, (*TestUser)(nil))
	assert.Nil(err)
	assert.Equals(count, 3)
}

func TestConnectionImpl_CountBy(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Insert multiple users
	cnx.Insert(ctx, &TestUser{Name: "User1", Email: "user1@example.com", Age: 20})
	cnx.Insert(ctx, &TestUser{Name: "User2", Email: "user2@example.com", Age: 25})
	cnx.Insert(ctx, &TestUser{Name: "User3", Email: "user3@example.com", Age: 30})

	// Count users with age >= 25
	count, err := cnx.CountBy(ctx, (*TestUser)(nil), "age >= ?", 25)
	assert.Nil(err)
	assert.Equals(count, 2)
}

func TestConnectionImpl_DeleteBy(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Insert multiple users
	cnx.Insert(ctx, &TestUser{Name: "User1", Email: "user1@example.com", Age: 20})
	cnx.Insert(ctx, &TestUser{Name: "User2", Email: "user2@example.com", Age: 25})
	cnx.Insert(ctx, &TestUser{Name: "User3", Email: "user3@example.com", Age: 30})

	// Delete users with age < 25
	err := cnx.DeleteBy(ctx, (*TestUser)(nil), "age < ?", 25)
	assert.Nil(err)

	// Verify deletion
	count, err := cnx.Count(ctx, (*TestUser)(nil))
	assert.Nil(err)
	assert.Equals(count, 2) // Only 2 users left
}

func TestConnectionImpl_UpdateBy(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Insert multiple users
	cnx.Insert(ctx, &TestUser{Name: "User1", Email: "user1@example.com", Age: 20})
	cnx.Insert(ctx, &TestUser{Name: "User2", Email: "user2@example.com", Age: 25})

	// Update age for all users with age < 25
	entity := &TestUser{Age: 99}
	rowsAffected, err := cnx.UpdateBy(ctx, entity, []string{"age"}, "age < ?", 25)
	assert.Nil(err)
	assert.Equals(rowsAffected, int64(1))

	// Verify update
	found := &TestUser{}
	cnx.FindBy(ctx, found, "email = ?", "user1@example.com")
	assert.Equals(found.Age, 99)
}

// ------------------------------------------------------------------------------------------------------------------
// Batch Operations Tests
// ------------------------------------------------------------------------------------------------------------------

func TestConnectionImpl_InsertBatch(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Insert multiple users in batch
	users := []*TestUser{
		{Name: "User1", Email: "user1@example.com", Age: 20},
		{Name: "User2", Email: "user2@example.com", Age: 25},
		{Name: "User3", Email: "user3@example.com", Age: 30},
	}
	err := cnx.InsertBatch(ctx, &users)
	assert.Nil(err)

	// Verify all inserted
	count, err := cnx.Count(ctx, (*TestUser)(nil))
	assert.Nil(err)
	assert.Equals(count, 3)
}

// ------------------------------------------------------------------------------------------------------------------
// Query Options Tests
// ------------------------------------------------------------------------------------------------------------------

func TestConnectionImpl_Query_WithLimit(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Insert multiple users
	for i := 1; i <= 5; i++ {
		cnx.Insert(ctx, &TestUser{
			Name:  "User" + string(rune(i)),
			Email: "user" + string(rune(i)) + "@example.com",
			Age:   20 + i,
		})
	}

	// Query with limit
	var users []*TestUser
	notFound, err := cnx.Query(ctx, &users, f.QueryOpts{Limit: 2})
	assert.Nil(err)
	assert.False(notFound)
	assert.Equals(len(users), 2)
}

func TestConnectionImpl_Query_WithOffset(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Insert users
	cnx.Insert(ctx, &TestUser{Name: "User1", Email: "user1@example.com", Age: 20})
	cnx.Insert(ctx, &TestUser{Name: "User2", Email: "user2@example.com", Age: 25})
	cnx.Insert(ctx, &TestUser{Name: "User3", Email: "user3@example.com", Age: 30})

	// Query with offset - SQLite requires LIMIT with OFFSET
	var users []*TestUser
	notFound, err := cnx.Query(ctx, &users, f.QueryOpts{
		Limit:   10, // Required for OFFSET to work
		Offset:  1,
		OrderBy: "id",
	})
	assert.Nil(err)
	assert.False(notFound)
	assert.Equals(len(users), 2) // Skip first one
}

func TestConnectionImpl_Query_WithWhere(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Insert users
	cnx.Insert(ctx, &TestUser{Name: "User1", Email: "user1@example.com", Age: 20})
	cnx.Insert(ctx, &TestUser{Name: "User2", Email: "user2@example.com", Age: 25})
	cnx.Insert(ctx, &TestUser{Name: "User3", Email: "user3@example.com", Age: 30})

	// Query with where clause
	var users []*TestUser
	notFound, err := cnx.Query(ctx, &users, f.QueryOpts{
		Where: "age >= ?",
		Args:  []any{25},
	})
	assert.Nil(err)
	assert.False(notFound)
	assert.Equals(len(users), 2)
}

func TestConnectionImpl_Query_WithOrderBy(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Insert users out of order
	cnx.Insert(ctx, &TestUser{Name: "User3", Email: "user3@example.com", Age: 30})
	cnx.Insert(ctx, &TestUser{Name: "User1", Email: "user1@example.com", Age: 20})
	cnx.Insert(ctx, &TestUser{Name: "User2", Email: "user2@example.com", Age: 25})

	// Query with order by
	var users []*TestUser
	notFound, err := cnx.Query(ctx, &users, f.QueryOpts{
		OrderBy: "age ASC",
	})
	assert.Nil(err)
	assert.False(notFound)
	assert.Equals(users[0].Age, 20)
	assert.Equals(users[1].Age, 25)
	assert.Equals(users[2].Age, 30)
}

// ------------------------------------------------------------------------------------------------------------------
// DatabaseUrl Tests
// ------------------------------------------------------------------------------------------------------------------

func TestConnectionImpl_DatabaseUrl(t *testing.T) {
	assert := test.NewAssertions(t)

	url := test.TestDatabaseURL()
	cnx, err := NewConnection(url)
	assert.Nil(err)

	// Should return the original URL
	assert.Equals(cnx.DatabaseUrl(), url)
}

// ------------------------------------------------------------------------------------------------------------------
// Edge Cases
// ------------------------------------------------------------------------------------------------------------------

func TestConnectionImpl_EmptyTableCount(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Count on empty table
	count, err := cnx.Count(ctx, (*TestUser)(nil))
	assert.Nil(err)
	assert.Equals(count, 0)
}

func TestConnectionImpl_DeleteNonExistent(t *testing.T) {
	assert := test.NewAssertions(t)
	ctx := context.Background()

	cnx := setupTestTable(t)

	// Try to delete non-existent user
	user := &TestUser{ID: 99999}
	err := cnx.Delete(ctx, user)
	// Should not error, just delete 0 rows
	assert.Nil(err)
}

// NOTE: Connection tests focus on SQLite in-memory database.
// PostgreSQL-specific features (schemas) are not tested here as they require
// a running PostgreSQL server.
//
// Features tested:
// - Basic CRUD (Insert, Update, Delete, Find)
// - Transactions (Begin, Commit, Rollback)
// - Batch operations
// - Query options (Limit, Offset, Where, OrderBy)
// - Counting and existence checks
// - Edge cases (empty tables, non-existent records)
//
// Features NOT tested (require external services):
// - PostgreSQL schema management
// - Database migrations (require migration files)
// - Join operations (require multiple tables)
