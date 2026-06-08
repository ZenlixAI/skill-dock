package persist

import (
	"context"
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
}
