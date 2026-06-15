package persist

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
	if storeSet.DB == nil {
		t.Fatal("expected shared *sql.DB to be initialized")
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
	if err := storeSet.DB.PingContext(ctx); err != nil {
		t.Fatalf("expected injected *sql.DB to remain usable, got %v", err)
	}
}
