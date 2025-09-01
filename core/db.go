package f

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

type Entity any
type ConnectionKey struct{}

type TenantEntity struct {
	Entity        `json:"-"`
	bun.BaseModel `bun:"table:tenants" `
	ID            *string    `bun:",pk" json:"id"`
	Name          *string    `json:"name"`
	ApiKey        string     `json:"api_key"`
	Slug          *string    `json:"slug"`
	Status        *string    `json:"status,omitempty"`
	DatabaseUrl   string     `json:"-"`
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

type Connection interface {
	DatabaseUrl() string
	//
	Tx(ctx context.Context) (Connection, error)
	Commit() error
	Rollback() error
	//
	FindBy(ctx context.Context, model Entity, where string, args ...any) (bool, error)
	ExistsBy(ctx context.Context, model Entity, where string, args ...any) (bool, error)
	Count(ctx context.Context, model Entity) (int, error)
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

func NewRepo(cnx Connection) Repo {
	return Repo{
		cnx: cnx,
	}
}

func ds(ctx context.Context) Connection {
	value := ctx.Value(ConnectionKey{})
	if value == nil {
		panic("MISSING_DS_IN_CONTEXT")
	}
	return value.(Connection)
}

func (r *Repo) Insert(ctx context.Context, entity Entity) error {
	return ds(ctx).Insert(ctx, entity)
}

func (r *Repo) InsertBatch(ctx context.Context, records Entity) error {
	if err := ds(ctx).InsertBatch(ctx, records); err != nil {
		return err
	}
	return nil
}

func (r *Repo) Update(ctx context.Context, entity ...any) error {
	for _, record := range entity {
		if err := ds(ctx).Update(ctx, record); err != nil {
			return err
		}
	}
	return nil

}

func (r *Repo) UpdateColumns(ctx context.Context, entity Entity, columns ...string) error {
	return ds(ctx).Update(ctx, entity, columns...)
}

func (r *Repo) Delete(ctx context.Context, entity ...Entity) error {
	for _, record := range entity {
		if err := ds(ctx).Delete(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repo) DeleteById(ctx context.Context, model Entity, id string) error {
	return ds(ctx).DeleteBy(ctx, model, "id=?", id)
}

func (r *Repo) FindBy(ctx context.Context, model Entity, where string, args ...any) (bool, error) {
	return ds(ctx).FindBy(ctx, model, where, args...)
}

func (r *Repo) ExistsBy(ctx context.Context, model Entity, where string, args ...any) (bool, error) {
	return ds(ctx).ExistsBy(ctx, model, where, args...)
}

func (r *Repo) CountAll(ctx context.Context, model Entity) (int, error) {
	return ds(ctx).Count(ctx, model)
}

func (r *Repo) CountBy(ctx context.Context, model Entity, where string, args ...any) (int, error) {
	return ds(ctx).CountBy(ctx, model, where, args...)
}

func (r *Repo) Query(ctx context.Context, model Entity, opts QueryOpts) (bool, error) {
	return ds(ctx).Query(ctx, model, opts)
}
