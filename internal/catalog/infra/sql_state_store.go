package infra

import (
	"database/sql"

	"skill_registry/business/catalog"
)

type SQLStateStore = catalog.SQLStateStore

func NewSQLStateStore(db *sql.DB, dialect string) *SQLStateStore {
	return catalog.NewSQLStateStore(db, dialect)
}
