package h

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"github.com/uptrace/bun/driver/pgdriver"
)

func ValidateDatabaseUrl(databaseUrl string) (bool, error) {
	var (
		sqldb *sql.DB
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

	} else {
		return false, fmt.Errorf("invalid database url: %s", databaseUrl)
	}

	err := sqldb.Ping()

	return err == nil, err
}
