package persist

import (
	"context"
	"database/sql"
	"time"

	platformpersist "skill_registry/internal/platform/persistence"
)

type SQLStoreSet = platformpersist.SQLStoreSet

var NewSQLStoreSet = platformpersist.NewSQLStoreSet

func NewSQLStoreSetWithDB(ctx context.Context, db *sql.DB, driver string, idempotencyTTL time.Duration) (*SQLStoreSet, error) {
	return platformpersist.NewSQLStoreSetWithDB(ctx, db, driver, idempotencyTTL)
}
