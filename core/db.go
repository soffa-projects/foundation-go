package f

import (
	"context"
	"io/fs"
)

type Entity any
type TenantCnx struct{}
type DefaultCnx struct{}

type Connection interface {
	DatabaseUrl() string
	//
	SetSchema(schema string) error
	Tx(ctx context.Context) (Connection, error)
	Commit() error
	Rollback() error
	Ping() error
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

type DBManager struct {
	ChangeLogTable string
	MigrationsFS   []fs.FS
}

type EntityID struct {
	ID string `bun:",pk" json:"id"`
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
	cnx           Connection
	Ctx           context.Context
	DefaultTenant bool
}

func NewRepo(cnx Connection) Repo {
	return Repo{
		cnx: cnx,
	}
}

func ds(ctx context.Context, defaultTenant bool) Connection {
	var value any
	if defaultTenant {
		value = ctx.Value(DefaultCnx{})
	} else {
		value = ctx.Value(TenantCnx{})
	}
	if value == nil {
		panic("MISSING_DS_IN_CONTEXT")
	}
	return value.(Connection)
}

func (r *Repo) Insert(ctx context.Context, entity Entity) error {
	return ds(ctx, r.DefaultTenant).Insert(ctx, entity)
}

func (r *Repo) InsertBatch(ctx context.Context, records Entity) error {
	if err := ds(ctx, r.DefaultTenant).InsertBatch(ctx, records); err != nil {
		return err
	}
	return nil
}

func (r *Repo) Update(ctx context.Context, entity ...any) error {
	for _, record := range entity {
		if err := ds(ctx, r.DefaultTenant).Update(ctx, record); err != nil {
			return err
		}
	}
	return nil

}

func (r *Repo) UpdateColumns(ctx context.Context, entity Entity, columns ...string) error {
	return ds(ctx, r.DefaultTenant).Update(ctx, entity, columns...)
}

func (r *Repo) Delete(ctx context.Context, entity ...Entity) error {
	for _, record := range entity {
		if err := ds(ctx, r.DefaultTenant).Delete(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repo) DeleteById(ctx context.Context, model Entity, id string) error {
	return ds(ctx, r.DefaultTenant).DeleteBy(ctx, model, "id=?", id)
}

func (r *Repo) FindBy(ctx context.Context, model Entity, where string, args ...any) (bool, error) {
	return ds(ctx, r.DefaultTenant).FindBy(ctx, model, where, args...)
}

func (r *Repo) ExistsBy(ctx context.Context, model Entity, where string, args ...any) (bool, error) {
	return ds(ctx, r.DefaultTenant).ExistsBy(ctx, model, where, args...)
}

func (r *Repo) CountAll(ctx context.Context, model Entity) (int, error) {
	return ds(ctx, r.DefaultTenant).Count(ctx, model)
}

func (r *Repo) CountBy(ctx context.Context, model Entity, where string, args ...any) (int, error) {
	return ds(ctx, r.DefaultTenant).CountBy(ctx, model, where, args...)
}

func (r *Repo) Query(ctx context.Context, model Entity, opts QueryOpts) (bool, error) {
	return ds(ctx, r.DefaultTenant).Query(ctx, model, opts)
}
