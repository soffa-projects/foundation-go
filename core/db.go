package f

import (
	"context"
	"database/sql"
	"io/fs"
	"time"

	"github.com/uptrace/bun"
)

type Entity any

// tenantIDKey is a custom type for context keys to avoid collisions
type TenantID struct{}
type DBIKey struct{}

type DataSource interface {
	CreateTenant(slug string, name string, databaseUrl string) (*TenantEntity, error)
	GetTenantList() ([]TenantEntity, error)
	FindByTenantById(id string) (*TenantEntity, error)
}

type TenantEntity struct {
	Entity        `json:"-"`
	bun.BaseModel `bun:"table:tenants" `
	ID            *string    `bun:",pk" json:"id"`
	Name          string     `json:"name"`
	Slug          string     `json:"slug"`
	Status        *string    `json:"status,omitempty"`
	DatabaseUrl   string     `json:"-"`
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

type DSOpt struct {
	MigrationsFS fs.FS
}

type TenantAlreadyExistsError struct {
	error
	Value string
}

// @deprecated
type DB interface {
	Tx(ctx context.Context, s *sql.TxOptions) (DB, error)
	// ------------------------------------------------------------------------------------------------------------------
	Commit() error
	Rollback() error
	FindBy(ctx context.Context, model Entity, where string, args ...any) (bool, error)
	ExistsBy(ctx context.Context, model Entity, where string, args ...any) (bool, error)
	CountBy(ctx context.Context, model Entity, where string, args ...any) (int, error)
	FindByJoin(ctx context.Context, model Entity, join string, where string, args ...any) (bool, error)
	CountByJoin(ctx context.Context, model Entity, join string, where string, args ...any) (int, error)
	Query(ctx context.Context, model Entity, opts QueryOpts) (bool, error)
	Insert(ctx context.Context, model Entity) error
	InsertBatch(ctx context.Context, models Entity) error
	Update(ctx context.Context, model Entity, columns ...string) error
	UpdateBy(ctx context.Context, entity Entity, columns []string, where string, args ...any) (int64, error)
	Delete(ctx context.Context, model Entity) error
	DeleteBy(ctx context.Context, model Entity, where string, args ...any) error
}

type Connection interface {
	DatabaseUrl() string
	/*DB

	//tenants  TenantLoader
	*/

	Commit() error
	Rollback() error
	FindBy(ctx context.Context, model Entity, where string, args ...any) (bool, error)
	ExistsBy(ctx context.Context, model Entity, where string, args ...any) (bool, error)
	CountBy(ctx context.Context, model Entity, where string, args ...any) (int, error)
	FindByJoin(ctx context.Context, model Entity, join string, where string, args ...any) (bool, error)
	CountByJoin(ctx context.Context, model Entity, join string, where string, args ...any) (int, error)
	Query(ctx context.Context, model Entity, opts QueryOpts) (bool, error)
	Insert(ctx context.Context, model Entity) error
	InsertBatch(ctx context.Context, models Entity) error
	Update(ctx context.Context, model Entity, columns ...string) error
	UpdateBy(ctx context.Context, entity Entity, columns []string, where string, args ...any) (int64, error)
	Delete(ctx context.Context, model Entity) error
	DeleteBy(ctx context.Context, model Entity, where string, args ...any) error
}

// ------------------------------------------------------------------------------------------------------------------
// TX IMPL
// ------------------------------------------------------------------------------------------------------------------

type QueryOpts struct {
	Columns string
	Joins   []string
	Where   string
	OrderBy string
	Args    []any
	Limit   int
	Offset  int
}

// ------------------------------------------------------------------------------------------------------------------
// REPOSITORY
// ------------------------------------------------------------------------------------------------------------------

type Repo struct {
	cnx Connection
	Ctx context.Context
}

func NewRepo(cnx Connection) *Repo {
	return &Repo{
		cnx: cnx,
	}
}

func (r *Repo) Insert(ctx context.Context, entity ...Entity) error {
	for _, record := range entity {
		if err := r.cnx.Insert(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repo) InsertBatch(ctx context.Context, records Entity) error {
	if err := r.cnx.InsertBatch(ctx, records); err != nil {
		return err
	}
	return nil
}

func (r *Repo) Update(ctx context.Context, entity ...any) error {
	for _, record := range entity {
		if err := r.cnx.Update(ctx, record); err != nil {
			return err
		}
	}
	return nil

}

func (r *Repo) UpdateColumns(ctx context.Context, entity Entity, columns ...string) error {
	return r.cnx.Update(ctx, entity, columns...)
}

func (r *Repo) Delete(ctx context.Context, entity ...Entity) error {
	for _, record := range entity {
		if err := r.cnx.Delete(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repo) DeleteById(ctx context.Context, model Entity, id string) error {
	return r.cnx.DeleteBy(ctx, model, "id=?", id)
}

func (r *Repo) FindBy(ctx context.Context, model Entity, where string, args ...any) (bool, error) {
	return r.cnx.FindBy(ctx, model, where, args...)
}

func (r *Repo) ExistsBy(ctx context.Context, model Entity, where string, args ...any) (bool, error) {
	return r.cnx.ExistsBy(ctx, model, where, args...)
}

func (r *Repo) CountBy(ctx context.Context, model Entity, where string, args ...any) (int, error) {
	return r.cnx.CountBy(ctx, model, where, args...)
}

func (r *Repo) Query(ctx context.Context, model Entity, opts QueryOpts) (bool, error) {
	return r.cnx.Query(ctx, model, opts)
}
