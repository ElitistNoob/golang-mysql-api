package db

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

func InitDatabase(dsn string) (*sql.DB, error) {
	return sql.Open("mysql", dsn)
}
