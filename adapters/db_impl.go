package adapters

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"strings"

	"github.com/pressly/goose/v3"
	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/driver/sqliteshim"
)

type connectionImpl struct {
	f.Connection

	Default     bool
	Url         string
	Id          string
	dialect     string
	db          bun.IDB
	schema      string
	initialized bool
	//tx          bool
}

func NewConnection(databaseUrl string) (f.Connection, error) {
	cnx := connectionImpl{
		Url:     databaseUrl,
		Default: true,
	}
	err := cnx.configure(nil, "")
	if err != nil {
		return nil, err
	}
	return cnx, nil
}

func (t connectionImpl) Ping() error {
	_, err := t.db.NewRaw("SELECT 1").Exec(context.Background())
	return err
}

func (t connectionImpl) Tx(ctx context.Context) (f.Connection, error) {
	if t.db == nil {
		return nil, errors.New("database not initialized")
	}
	_, err := t.db.BeginTx(ctx, &sql.TxOptions{
		ReadOnly:  false,
		Isolation: sql.LevelDefault,
	})
	if t.schema != "" {
		t.SetSchema(t.schema)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	return connectionImpl{
		Default:     t.Default,
		Url:         t.Url,
		Id:          fmt.Sprintf("%s-tx-%s", t.Id, h.RandomString(5)),
		dialect:     t.dialect,
		db:          t.db,
		schema:      t.schema,
		initialized: t.initialized,
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

func (t *connectionImpl) configure(migrationsFS []fs.FS, prefix string) error {
	var (
		sqldb   *sql.DB
		db      *bun.DB
		err     error
		dialect string
	)

	if strings.HasPrefix(t.Url, "postgres://") || strings.HasPrefix(t.Url, "postgresql://") {
		u, err := url.Parse(t.Url)
		if err != nil {
			return err
		}
		t.schema = u.Query().Get("schema")
		if t.schema != "" {
			t.Url, err = h.RemoveParamFromUrl(t.Url, "schema")
			if err != nil {
				return err
			}
		}

		sqldb = sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(t.Url)))
		db = bun.NewDB(sqldb, pgdialect.New())
		dialect = "postgres"

	} else if strings.HasPrefix(t.Url, "sqlite://") {
		dialect = "sqlite3"
		sqliteDSN := strings.Replace(t.Url, "sqlite://", "", 1)
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

	var paths []string
	if t.Default {
		paths = []string{"resources/db/migrations/shared", "db/migrations/shared"}
	} else {
		paths = []string{"resources/db/migrations/tenant", "db/migrations/tenant"}
	}
	changeLogTable := "database_changelog"
	if prefix != "" {
		changeLogTable = fmt.Sprintf("%s_%s", strings.TrimSuffix(prefix, "_"), changeLogTable)
	}
	if len(migrationsFS) > 0 {
		for _, dir := range migrationsFS {
			for _, path := range paths {
				if err := t.migrate(changeLogTable, dir, path); err != nil {
					return err
				}
			}
		}
	}

	if t.schema != "" {
		err := t.SetSchema(t.schema)
		if err != nil {
			return err
		}
	}
	t.initialized = true
	return nil
}

func (t connectionImpl) migrate(changeLogTable string, dir fs.FS, path string) error {

	if dir == nil {
		return nil
	}

	// Check if there are any *.sql files in the migration directory
	entries, err := fs.ReadDir(dir, path)
	if err != nil {
		return nil
	}
	if len(entries) == 0 {
		return nil
	}
	hasSQL := false
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			hasSQL = true
			break
		}
	}
	if !hasSQL {
		return nil
	}

	goose.SetBaseFS(dir)
	ctx := context.Background()

	if err := goose.SetDialect(t.dialect); err != nil {
		log.Fatal("failed to set dialect: %v", err)
	}
	goose.SetTableName(changeLogTable)

	if t.dialect == "postgres" {
		schema := "public"
		if t.schema != "" {
			schema = t.schema
		}
		if _, err := t.db.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)); err != nil {
			return fmt.Errorf("failed to create schema %s: %v", schema, err)
		}
		if _, err := t.db.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schema)); err != nil {
			return fmt.Errorf("failed to set search path %s: %v", schema, err)
		}
	}
	if err := goose.Up((t.db.(*bun.DB)).DB, path, goose.WithAllowMissing()); err != nil {
		return fmt.Errorf("failed to run migrations for %s: %v", t.Id, err)
	}

	log.Info("migrations completed for tenant: %s", t.Id)
	return nil
}

func (t connectionImpl) Insert(ctx context.Context, entity f.Entity) error {
	_, err := t.db.NewInsert().Model(entity).Exec(ctx)
	return err
}

func (t connectionImpl) InsertBatch(ctx context.Context, entities f.Entity) error {
	_, err := t.db.NewInsert().Model(entities).Exec(ctx)
	return err
}

func (t connectionImpl) SetSchema(schema string) error {
	if t.dialect == "postgres" {
		if _, err := t.db.(*bun.DB).Exec(fmt.Sprintf("SET search_path TO %s", bun.Ident(schema))); err != nil {
			return fmt.Errorf("failed to set search path %s: %v", schema, err)
		}
	}
	return nil
}

func (t connectionImpl) Update(ctx context.Context, entity f.Entity, columns ...string) error {
	_, err := t.db.
		NewUpdate().
		Model(entity).
		Column(columns...).
		WherePK().
		Exec(ctx)
	return err
}

func (t connectionImpl) UpdateBy(ctx context.Context, entity f.Entity, columns []string, where string, args ...any) (int64, error) {
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

func (t connectionImpl) Delete(ctx context.Context, entity f.Entity) error {
	_, err := t.db.NewDelete().Model(entity).WherePK().Exec(ctx)
	return err
}

func (t connectionImpl) FindBy(ctx context.Context, entity f.Entity, where string, args ...any) (bool, error) {
	err := t.db.NewSelect().Model(entity).Where(where, args...).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (t connectionImpl) ExistsBy(ctx context.Context, entity f.Entity, where string, args ...any) (bool, error) {
	err := t.db.NewSelect().Model(entity).Where(where, args...).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (t connectionImpl) Count(ctx context.Context, entity f.Entity) (int, error) {
	count, err := t.db.NewSelect().
		Model(entity).
		Count(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
	}
	return count, err
}

func (t connectionImpl) CountBy(ctx context.Context, entity f.Entity, where string, args ...any) (int, error) {
	return countByJoin(ctx, t.db.NewSelect(), entity, "", where, args...)
}

func (t connectionImpl) DeleteBy(ctx context.Context, entity f.Entity, where string, args ...any) error {
	_, err := t.db.NewDelete().Model(entity).Where(where, args...).Exec(ctx)
	return err
}

func (t connectionImpl) FindByJoin(ctx context.Context, model f.Entity, join string, where string, args ...any) (bool, error) {
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

func (t connectionImpl) Query(ctx context.Context, model f.Entity, opts ...f.QueryOpts) (bool, error) {
	return Query(ctx, t.db.NewSelect(), model, opts...)
}

func (t connectionImpl) CountByJoin(ctx context.Context, model f.Entity, join string, where string, args ...any) (int, error) {
	return countByJoin(ctx, t.db.NewSelect(), model, join, where, args...)
}

func (t connectionImpl) DatabaseUrl() string {
	return t.Url
}

// ------------------------------------------------------------------------------------------------------------------
// COMMON
// ------------------------------------------------------------------------------------------------------------------

func countByJoin(ctx context.Context, query *bun.SelectQuery, model f.Entity, join string, where string, args ...any) (int, error) {
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

func Query(ctx context.Context, query *bun.SelectQuery, model f.Entity, options ...f.QueryOpts) (bool, error) {
	q := query.Model(model)
	for _, opts := range options {
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
	}
	if err := q.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}
