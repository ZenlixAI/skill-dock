package persistence

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestNewSQLStoreSetSharesSingleDBAcrossStores(t *testing.T) {
	ctx := context.Background()
	databaseURL := "file:" + filepath.Join(t.TempDir(), "stores.db")

	storeSet, err := NewSQLStoreSet(ctx, databaseURL, time.Hour)
	if err != nil {
		t.Fatalf("create store set: %v", err)
	}

	if storeSet.Idempotency == nil || storeSet.Workspace == nil || storeSet.Audit == nil {
		t.Fatalf("expected all stores to be initialized, got %#v", storeSet)
	}
	if storeSet.Idempotency.db != storeSet.Workspace.db || storeSet.Idempotency.db != storeSet.Audit.db {
		t.Fatal("expected idempotency, workspace, and audit stores to share one *sql.DB")
	}
	if storeSet.Idempotency.dialect != storeSet.Workspace.dialect {
		t.Fatalf("expected store dialects to match, got %q and %q", storeSet.Idempotency.dialect, storeSet.Workspace.dialect)
	}
	if storeSet.Audit.dialect != auditSQLDialectForDriver(storeSet.Idempotency.dialect) {
		t.Fatalf("expected audit dialect to match translated driver dialect, got %q", storeSet.Audit.dialect)
	}
}

func TestNewSQLStoreSetWithDBReusesInjectedHandle(t *testing.T) {
	ctx := context.Background()
	databaseURL := "file:" + filepath.Join(t.TempDir(), "stores.db")
	db, err := sql.Open(sqlDriverNameForURL(databaseURL), databaseURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	storeSet, err := NewSQLStoreSetWithDB(ctx, db, sqlDriverNameForURL(databaseURL), time.Hour)
	if err != nil {
		t.Fatalf("create store set with injected db: %v", err)
	}
	if storeSet.DB != db {
		t.Fatal("expected store set to reuse injected *sql.DB")
	}
	if storeSet.Idempotency.db != db || storeSet.Workspace.db != db || storeSet.Audit.db != db {
		t.Fatal("expected all stores to reuse injected *sql.DB")
	}
}
