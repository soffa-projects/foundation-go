package f

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"strings"

	"github.com/pressly/goose/v3"
	"github.com/soffa-projects/foundation-go/log"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/driver/sqliteshim"
)

type connectionImpl struct {
	Connection

	Default     bool
	Url         string
	Id          string
	dialect     string
	db          bun.IDB
	schema      string
	initialized bool
	//tx          bool
}

func (t connectionImpl) Tx(ctx context.Context) (Connection, error) {
	if t.db == nil {
		return nil, errors.New("database not initialized")
	}
	tx, err := t.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	return &connectionImpl{
		Default: t.Default,
		Url:     t.Url,
		Id:      t.Id,
		dialect: t.dialect,
		db:      tx,
	}, nil
}

func (t connectionImpl) Commit() error {
	if tx, ok := t.db.(bun.Tx); ok {
		return tx.Commit()
	}
	return nil
}

func (t connectionImpl) Rollback() error {
	if tx, ok := t.db.(bun.Tx); ok {
		return tx.Rollback()
	}
	return nil
}

func (t *connectionImpl) configure(dir fs.FS) error {
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
			log.Fatal("failed to open SQLite database: %v", err)
		}
		db = bun.NewDB(sqldb, sqlitedialect.New())
		_, err = db.Exec("PRAGMA foreign_keys = ON;")
		if err != nil {
			log.Fatal("failed to enable foreign keys: %v", err)
		}
	}
	t.db = db
	t.dialect = dialect
	t.schema = u.Query().Get("schema")

	return t.migrate(dir)
}

func (t connectionImpl) migrate(dir fs.FS) error {
	log.Info("migrating tenant %s", t.Id)
	goose.SetBaseFS(dir)
	ctx := context.Background()
	var path string
	if t.Default {
		path = "db/migrations/shared"
	} else {
		path = "db/migrations/tenant"
	}
	if err := goose.SetDialect(t.dialect); err != nil {
		log.Fatal("failed to set dialect: %v", err)
	}
	goose.SetTableName("database_changelog")
	goose.SetBaseFS(dir)

	log.Info("running migrations for %s", t.Id)
	if t.dialect == "postgres" {
		schema := "public"
		if t.schema != "" {
			schema = t.schema
		}
		if _, err := t.db.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)); err != nil {
			return fmt.Errorf("failed to create schema: %v", err)
		}
		if _, err := t.db.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema)); err != nil {
			return fmt.Errorf("failed to set search path: %v", err)
		}
	}
	if err := goose.Up((t.db.(*bun.DB)).DB, path, goose.WithAllowMissing()); err != nil {
		return fmt.Errorf("failed to run migrations: %v", err)
	}
	return nil
}

/*
func (i connectionImpl) Tx(ctx context.Context, s *sql.TxOptions) (DB, error) {
	tx, err := i.db.BeginTx(ctx, s)
	if err != nil {
		return nil, err
	}
	return txImpl{
		tx: tx,
	}, nil
}
*/

func (t connectionImpl) Insert(ctx context.Context, entity Entity) error {
	_, err := t.db.NewInsert().Model(entity).Exec(ctx)
	return err
}

func (t connectionImpl) InsertBatch(ctx context.Context, entities Entity) error {
	_, err := t.db.NewInsert().Model(entities).Exec(ctx)
	return err
}

func (t connectionImpl) Update(ctx context.Context, entity Entity, columns ...string) error {
	_, err := t.db.
		NewUpdate().
		Model(entity).
		Column(columns...).
		WherePK().
		Exec(ctx)
	return err
}

func (t connectionImpl) UpdateBy(ctx context.Context, entity Entity, columns []string, where string, args ...any) (int64, error) {
	res, err := t.db.
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

func (t connectionImpl) Delete(ctx context.Context, entity Entity) error {
	_, err := t.db.NewDelete().Model(entity).WherePK().Exec(ctx)
	return err
}

func (t connectionImpl) FindBy(ctx context.Context, entity Entity, where string, args ...any) (bool, error) {
	err := t.db.NewSelect().Model(entity).Where(where, args...).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (t connectionImpl) ExistsBy(ctx context.Context, entity Entity, where string, args ...any) (bool, error) {
	err := t.db.NewSelect().Model(entity).Where(where, args...).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (t connectionImpl) CountBy(ctx context.Context, entity Entity, where string, args ...any) (int, error) {
	return countByJoin(ctx, t.db.NewSelect(), entity, "", where, args...)
}

func (t connectionImpl) DeleteBy(ctx context.Context, entity Entity, where string, args ...any) error {
	_, err := t.db.NewDelete().Model(entity).Where(where, args...).Exec(ctx)
	return err
}

func (t connectionImpl) FindByJoin(ctx context.Context, model Entity, join string, where string, args ...any) (bool, error) {
	err := t.db.NewSelect().
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

func (t connectionImpl) Query(ctx context.Context, model Entity, opts QueryOpts) (bool, error) {
	return Query(ctx, t.db.NewSelect(), model, opts)
}

func (t connectionImpl) CountByJoin(ctx context.Context, model Entity, join string, where string, args ...any) (int, error) {
	return countByJoin(ctx, t.db.NewSelect(), model, join, where, args...)
}

func (t connectionImpl) DatabaseUrl() string {
	return t.Url
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
