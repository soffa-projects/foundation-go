package micro

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"strings"

	"github.com/pressly/goose/v3"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/driver/sqliteshim"
)

type Entity any

type EntityManager struct {
	DefaultTenant *DBI
	TenantLoader  TenantLoader
	//DB DB
	MigrationsFS fs.FS
	tenants      map[string]*DBI
}

func createEntityManager(cfg DatabaseConfig) EntityManager {
	defaultTenant := &DBI{
		Id:      "public",
		Default: true,
		Url:     cfg.DatabaseURL,
	}
	if err := defaultTenant.configure(cfg.MigrationsFS); err != nil {
		log.Fatalf("failed to run migrations for default tenant: %v", err)
	}
	return EntityManager{
		DefaultTenant: defaultTenant,
		TenantLoader:  cfg.TenatLoader,
	}
}

func (e *EntityManager) Migrate(tenant string) error {
	dbi, err := e.Get(tenant)
	if err != nil {
		return err
	}
	return dbi.migrate(e.MigrationsFS)
}

func (e *EntityManager) Get(tenant string) (*DBI, error) {
	if tenant == "default" || tenant == "" || tenant == "public" {
		return e.DefaultTenant, nil
	}
	if tenantInfo, ok := e.tenants[tenant]; ok {
		return tenantInfo, nil
	}
	tenantInfo, err := e.TenantLoader.Get(e, tenant)
	if err != nil {
		return nil, err
	}

	dbi := &DBI{
		Id:      tenantInfo.Id,
		Default: false,
		Url:     tenantInfo.Url,
	}
	if err := dbi.configure(e.MigrationsFS); err != nil {
		return nil, err
	}
	e.tenants[tenant] = dbi
	return dbi, nil
}

/*
func parseDatabase(databaseUrl string) (*DBI, error) {

}
*/

type DB interface {
	Tx(ctx context.Context, s *sql.TxOptions) (DB, error)

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

type DBI struct {
	DB
	Default bool
	Url     string
	Id      string
	dialect string
	db      *bun.DB
	schema  string
	//tenants  TenantLoader
}

func (t *DBI) configure(dir fs.FS) error {
	var (
		sqldb   *sql.DB
		db      *bun.DB
		err     error
		dialect string
	)

	u, err := url.Parse(t.Url)
	if err != nil {
		return err
	}

	if strings.HasPrefix(t.Url, "postgres://") || strings.HasPrefix(t.Url, "postgresql://") {
		sqldb = sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(t.Url)))
		db = bun.NewDB(sqldb, pgdialect.New())
		dialect = "postgres"
	} else if strings.HasPrefix(t.Url, "sqlite://") {
		dialect = "sqlite3"
		sqliteDSN := "file::memory:?cache=shared"
		sqldb, err = sql.Open(sqliteshim.ShimName, sqliteDSN)
		if err != nil {
			log.Fatalf("failed to open SQLite database: %v", err)
		}
		db = bun.NewDB(sqldb, sqlitedialect.New())
		_ = R(db.Exec("PRAGMA foreign_keys = ON;"))
	}
	t.db = db
	t.dialect = dialect
	t.schema = u.Query().Get("schema")

	return t.migrate(dir)
}

func (t *DBI) migrate(dir fs.FS) error {
	log.Infof("migrating tenant %s", t.Id)
	goose.SetBaseFS(dir)
	var path string
	if t.Default {
		path = "db/migrations/shared"
	} else {
		path = "db/migrations/tenant"
	}
	if err := goose.SetDialect(t.dialect); err != nil {
		log.Fatalf("failed to set dialect: %v", err)
	}
	goose.SetTableName("database_changelog")
	goose.SetBaseFS(dir)

	log.Infof("running migrations for %s", t.Id)
	if t.dialect == "postgres" {
		schema := "public"
		if t.schema != "" {
			schema = t.schema
		}
		if _, err := t.db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)); err != nil {
			return fmt.Errorf("failed to create schema: %v", err)
		}
		if _, err := t.db.Exec(fmt.Sprintf("SET search_path TO %s", schema)); err != nil {
			return fmt.Errorf("failed to set search path: %v", err)
		}
	}
	if err := goose.Up(t.db.DB, path, goose.WithAllowMissing()); err != nil {
		return fmt.Errorf("failed to run migrations: %v", err)
	}
	return nil
}

func (i DBI) Tx(ctx context.Context, s *sql.TxOptions) (DB, error) {
	tx, err := i.db.BeginTx(ctx, s)
	if err != nil {
		return nil, err
	}
	return txImpl{
		tx: tx,
	}, nil
}

func (t DBI) Insert(ctx context.Context, entity Entity) error {
	_, err := prepareDB(ctx).NewInsert().Model(entity).Exec(ctx)
	return err
}

func (t DBI) InsertBatch(ctx context.Context, entities Entity) error {
	_, err := prepareDB(ctx).NewInsert().Model(entities).Exec(ctx)
	return err
}

func (t DBI) Update(ctx context.Context, entity Entity, columns ...string) error {
	_, err := prepareDB(ctx).
		NewUpdate().
		Model(entity).
		Column(columns...).
		WherePK().
		Exec(ctx)
	return err
}

func (t DBI) UpdateBy(ctx context.Context, entity Entity, columns []string, where string, args ...any) (int64, error) {
	res, err := prepareDB(ctx).
		NewUpdate().
		Model(entity).
		Column(columns...).
		Where(where, args...).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (t DBI) Delete(ctx context.Context, entity Entity) error {
	_, err := prepareDB(ctx).NewDelete().Model(entity).WherePK().Exec(ctx)
	return err
}

func (t DBI) FindBy(ctx context.Context, entity Entity, where string, args ...any) (bool, error) {
	err := prepareDB(ctx).NewSelect().Model(entity).Where(where, args...).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (t DBI) ExistsBy(ctx context.Context, entity Entity, where string, args ...any) (bool, error) {
	err := prepareDB(ctx).NewSelect().Model(entity).Where(where, args...).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (t DBI) CountBy(ctx context.Context, entity Entity, where string, args ...any) (int, error) {
	return countByJoin(ctx, prepareDB(ctx).NewSelect(), entity, "", where, args...)
}

func (t DBI) DeleteBy(ctx context.Context, entity Entity, where string, args ...any) error {
	_, err := prepareDB(ctx).NewDelete().Model(entity).Where(where, args...).Exec(ctx)
	return err
}

func (t DBI) FindByJoin(ctx context.Context, model Entity, join string, where string, args ...any) (bool, error) {
	err := prepareDB(ctx).NewSelect().
		Model(model).
		Join(join).
		Where(where, args...).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (t DBI) Query(ctx context.Context, model Entity, opts QueryOpts) (bool, error) {
	return Query(ctx, prepareDB(ctx).NewSelect(), model, opts)
}

func (t DBI) CountByJoin(ctx context.Context, model Entity, join string, where string, args ...any) (int, error) {
	return countByJoin(ctx, prepareDB(ctx).NewSelect(), model, join, where, args...)
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

type txImpl struct {
	DB
	tx bun.Tx
}

func (t txImpl) Commit() error {
	return t.tx.Commit()
}

func (t txImpl) Rollback() error {
	return t.tx.Commit()
}

func prepareDB(ctx context.Context) *bun.DB {
	dbi := ctx.Value(DBIKey{})
	if dbi == nil {
		return nil
	}
	return dbi.(*DBI).db
}

func prepareTx(ctx context.Context) *bun.Tx {
	tx := ctx.Value(DBIKey{})
	if tx == nil {
		return nil
	}
	return tx.(*bun.Tx)
}

func (t txImpl) Insert(ctx context.Context, entity Entity) error {
	_, err := prepareTx(ctx).NewInsert().Model(entity).Exec(ctx)
	return err
}

func (t txImpl) InsertBatch(ctx context.Context, entities Entity) error {
	_, err := prepareTx(ctx).NewInsert().Model(entities).Exec(ctx)
	return err
}

func (t txImpl) Update(ctx context.Context, entity Entity, columns ...string) error {
	_, err := prepareTx(ctx).
		NewUpdate().
		Model(entity).
		Column(columns...).
		WherePK().
		Exec(ctx)
	return err
}

func (t txImpl) UpdateBy(ctx context.Context, entity Entity, columns []string, where string, args ...any) (int64, error) {
	res, err := prepareTx(ctx).
		NewUpdate().
		Model(entity).
		Column(columns...).
		Where(where, args...).
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (t txImpl) Delete(ctx context.Context, entity Entity) error {
	_, err := prepareTx(ctx).NewDelete().Model(entity).WherePK().Exec(ctx)
	return err
}

func (t txImpl) FindBy(ctx context.Context, entity Entity, where string, args ...any) (bool, error) {
	err := prepareTx(ctx).NewSelect().Model(entity).Where(where, args...).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (t txImpl) ExistsBy(ctx context.Context, entity Entity, where string, args ...any) (bool, error) {

	err := prepareTx(ctx).NewSelect().Model(entity).Where(where, args...).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (t txImpl) CountBy(ctx context.Context, entity Entity, where string, args ...any) (int, error) {
	return countByJoin(ctx, prepareTx(ctx).NewSelect(), entity, "", where, args...)
}

func (t txImpl) DeleteBy(ctx context.Context, entity Entity, where string, args ...any) error {
	_, err := prepareTx(ctx).NewDelete().Model(entity).Where(where, args...).Exec(ctx)
	return err
}

func (t txImpl) FindByJoin(ctx context.Context, model Entity, join string, where string, args ...any) (bool, error) {
	err := prepareTx(ctx).NewSelect().
		Model(model).
		Join(join).
		Where(where, args...).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (t txImpl) Query(ctx context.Context, model Entity, opts QueryOpts) (bool, error) {
	return Query(ctx, prepareTx(ctx).NewSelect(), model, opts)
}

func (t txImpl) CountByJoin(ctx context.Context, model Entity, join string, where string, args ...any) (int, error) {
	return countByJoin(ctx, prepareTx(ctx).NewSelect(), model, join, where, args...)
}

// ------------------------------------------------------------------------------------------------------------------
// REPOSITORY
// ------------------------------------------------------------------------------------------------------------------

type Repo struct {
	//DB
	Ctx context.Context
}

func (r *Repo) Insert(ctx context.Context, entity ...Entity) error {
	for _, record := range entity {
		if err := getDB(ctx).Insert(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repo) InsertBatch(ctx context.Context, records Entity) error {
	if err := getDB(ctx).InsertBatch(ctx, records); err != nil {
		return err
	}
	return nil
}

func (r *Repo) Update(ctx context.Context, entity ...any) error {
	for _, record := range entity {
		if err := getDB(ctx).Update(ctx, record); err != nil {
			return err
		}
	}
	return nil

}

func (r *Repo) UpdateColumns(ctx context.Context, entity Entity, columns ...string) error {
	return getDB(ctx).Update(ctx, entity, columns...)
}

func (r *Repo) Delete(ctx context.Context, entity ...Entity) error {
	for _, record := range entity {
		if err := getDB(ctx).Delete(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repo) DeleteById(ctx context.Context, model Entity, id string) error {
	return getDB(ctx).DeleteBy(ctx, model, "id=?", id)
}

func (r *Repo) FindBy(ctx context.Context, model Entity, where string, args ...any) (bool, error) {
	return getDB(ctx).FindBy(ctx, model, where, args...)
}

func (r *Repo) ExistsBy(ctx context.Context, model Entity, where string, args ...any) (bool, error) {
	return getDB(ctx).ExistsBy(ctx, model, where, args...)
}

func (r *Repo) CountBy(ctx context.Context, model Entity, where string, args ...any) (int, error) {
	return getDB(ctx).CountBy(ctx, model, where, args...)
}

func (r *Repo) Query(ctx context.Context, model Entity, opts QueryOpts) (bool, error) {
	return getDB(ctx).Query(ctx, model, opts)
}

func getDB(ctx context.Context) DB {
	dbi := ctx.Value(DBIKey{})
	if dbi == nil {
		LogError("no db found in context")
		return nil
	}
	return (dbi.(*DBI))
}

// ------------------------------------------------------------------------------------------------------------------
// COMMON
// ------------------------------------------------------------------------------------------------------------------

func countByJoin(ctx context.Context, query *bun.SelectQuery, model Entity, join string, where string, args ...any) (int, error) {
	count, err := query.
		Model(model).
		Join(join).
		Where(where, args...).
		Count(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
	}
	return count, err
}

func Query(ctx context.Context, query *bun.SelectQuery, model Entity, opts QueryOpts) (bool, error) {
	q := query.NewSelect().Model(model)
	if opts.Columns != "" {
		q = q.ColumnExpr(opts.Columns)
	}
	if len(opts.Joins) > 0 {
		for _, join := range opts.Joins {
			q = q.Join(join)
		}
	}
	if opts.Where != "" {
		q = q.Where(opts.Where, opts.Args...)
	}
	if opts.OrderBy != "" {
		q = q.Order(opts.OrderBy)
	}
	if opts.Limit > 0 {
		q = q.Limit(opts.Limit)
	}
	if opts.Offset > 0 {
		q = q.Offset(opts.Offset)
	}
	if err := q.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}
