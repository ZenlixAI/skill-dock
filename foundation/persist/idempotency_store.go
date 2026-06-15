package persist

import (
	"database/sql"
	"time"

	platformpersist "skill_registry/internal/platform/persistence"
)

var ErrIdempotencyRecordNotFound = platformpersist.ErrIdempotencyRecordNotFound

type IdempotencyStore = platformpersist.IdempotencyStore
type SQLIdempotencyStore = platformpersist.SQLIdempotencyStore
type IdempotencyRecord = platformpersist.IdempotencyRecord

var NewSQLIdempotencyStore = platformpersist.NewSQLIdempotencyStore

func newSQLIdempotencyStore(db *sql.DB, dialect string, ttl time.Duration) *SQLIdempotencyStore {
	return platformpersist.NewSQLIdempotencyStoreWithDB(db, dialect, ttl)
}

func sqlDriverNameForURL(databaseURL string) string {
	return platformpersist.SQLDriverNameForURL(databaseURL)
}

func bindSQL(dialect string, query string) string {
	return platformpersist.BindSQL(dialect, query)
}
