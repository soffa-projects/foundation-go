package h

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/driver/sqliteshim"
)

func ValidateDatabaseUrl(databaseUrl string) (bool, error) {
	var (
		sqldb *sql.DB
		err   error
	)
	schema := ""

	if strings.HasPrefix(databaseUrl, "postgres://") || strings.HasPrefix(databaseUrl, "postgresql://") {
		u, err := url.Parse(databaseUrl)
		if err != nil {
			return false, err
		}
		schema = u.Query().Get("schema")
		if schema != "" {
			databaseUrl, err = RemoveParamFromUrl(databaseUrl, "schema")
			if err != nil {
				return false, err
			}
		}

		sqldb = sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(databaseUrl)))

	} else if strings.HasPrefix(databaseUrl, "sqlite://") {
		sqliteDSN := strings.Replace(databaseUrl, "sqlite://", "", 1)
		sqldb, err = sql.Open(sqliteshim.ShimName, sqliteDSN)
		if err != nil {
			return false, fmt.Errorf("failed to open SQLite database: %v", err)
		}
	} else {
		return false, fmt.Errorf("invalid database url: %s", databaseUrl)
	}

	// FIXED: Close the database connection to prevent resource leak
	defer sqldb.Close()

	err = sqldb.Ping()

	return err == nil, err
}
