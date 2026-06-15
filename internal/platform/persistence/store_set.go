package persistence

import (
	"context"
	"database/sql"
	"time"
)

type SQLStoreSet struct {
	DB          *sql.DB
	Idempotency *SQLIdempotencyStore
	Workspace   *SQLWorkspaceStore
	Audit       *SQLAuditStore
}

func NewSQLStoreSet(ctx context.Context, databaseURL string, idempotencyTTL time.Duration) (*SQLStoreSet, error) {
	driver := sqlDriverNameForURL(databaseURL)
	db, err := sql.Open(driver, databaseURL)
	if err != nil {
		return nil, err
	}
	storeSet, err := NewSQLStoreSetWithDB(ctx, db, driver, idempotencyTTL)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return storeSet, nil
}

func NewSQLStoreSetWithDB(ctx context.Context, db *sql.DB, driver string, idempotencyTTL time.Duration) (*SQLStoreSet, error) {
	storeSet := &SQLStoreSet{
		DB:          db,
		Idempotency: newSQLIdempotencyStore(db, driver, idempotencyTTL),
		Workspace:   newSQLWorkspaceStore(db, driver),
		Audit:       newSQLAuditStore(db, auditSQLDialectForDriver(driver)),
	}

	if err := storeSet.Idempotency.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := storeSet.Workspace.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := storeSet.Audit.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return storeSet, nil
}
