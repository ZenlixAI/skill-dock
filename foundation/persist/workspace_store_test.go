package persist

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWorkspaceMigrationStatementsUsePostgresTypes(t *testing.T) {
	statements := workspaceMigrationStatements("pgx")
	if len(statements) == 0 {
		t.Fatal("expected migration statements")
	}

	createTable := statements[0]
	if strings.Contains(createTable, "BLOB") || strings.Contains(createTable, "BYTEA") {
		t.Fatalf("expected postgres migration to use managed path columns, got %q", createTable)
	}
	if !strings.Contains(createTable, "workspace_path TEXT NOT NULL") {
		t.Fatalf("expected workspace_path TEXT NOT NULL, got %q", createTable)
	}
	if !strings.Contains(createTable, "bundle_path TEXT") {
		t.Fatalf("expected bundle_path TEXT, got %q", createTable)
	}
}

func TestWorkspaceMigrationStatementsUseSQLiteTypes(t *testing.T) {
	statements := workspaceMigrationStatements("sqlite")
	if len(statements) == 0 {
		t.Fatal("expected migration statements")
	}

	createTable := statements[0]
	if strings.Contains(createTable, "BLOB") || strings.Contains(createTable, "BYTEA") {
		t.Fatalf("expected sqlite migration to avoid blob columns, got %q", createTable)
	}
	if !strings.Contains(createTable, "workspace_path TEXT NOT NULL") {
		t.Fatalf("expected sqlite migration to use workspace_path TEXT NOT NULL, got %q", createTable)
	}
	if !strings.Contains(createTable, "bundle_path TEXT") {
		t.Fatalf("expected sqlite migration to use bundle_path TEXT, got %q", createTable)
	}
}

func TestSQLWorkspaceStoreRoundTripsManagedPaths(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "workspace-store.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	store := newSQLWorkspaceStore(db, "sqlite")
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate workspace store: %v", err)
	}

	createdAt := time.Date(2026, time.June, 8, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(15 * time.Minute)

	if err := store.SaveWorkspace(ctx, "ws-123", "/tmp/workspaces/ws-123", createdAt); err != nil {
		t.Fatalf("save workspace: %v", err)
	}

	record, err := store.LoadWorkspace(ctx, "ws-123")
	if err != nil {
		t.Fatalf("load workspace: %v", err)
	}
	if record.WorkspaceID != "ws-123" {
		t.Fatalf("expected workspace ID ws-123, got %q", record.WorkspaceID)
	}
	if record.WorkspacePath != "/tmp/workspaces/ws-123" {
		t.Fatalf("expected workspace path to round-trip, got %q", record.WorkspacePath)
	}
	if record.BundlePath != "" {
		t.Fatalf("expected empty bundle path before bundle save, got %q", record.BundlePath)
	}
	if !record.CreatedAt.Equal(createdAt) {
		t.Fatalf("expected created_at %s, got %s", createdAt, record.CreatedAt)
	}
	if !record.UpdatedAt.Equal(createdAt) {
		t.Fatalf("expected updated_at %s after initial save, got %s", createdAt, record.UpdatedAt)
	}

	_, err = store.LoadBundle(ctx, "ws-123")
	if !errors.Is(err, ErrWorkspaceBundleNotFound) {
		t.Fatalf("expected ErrWorkspaceBundleNotFound before bundle save, got %v", err)
	}

	if err := store.SaveBundle(ctx, "ws-123", "/tmp/bundles/ws-123.tgz", updatedAt); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	record, err = store.LoadWorkspace(ctx, "ws-123")
	if err != nil {
		t.Fatalf("load workspace after bundle save: %v", err)
	}
	if record.BundlePath != "/tmp/bundles/ws-123.tgz" {
		t.Fatalf("expected bundle path to round-trip in workspace record, got %q", record.BundlePath)
	}
	if !record.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("expected updated_at %s after bundle save, got %s", updatedAt, record.UpdatedAt)
	}

	bundlePath, err := store.LoadBundle(ctx, "ws-123")
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	if bundlePath != "/tmp/bundles/ws-123.tgz" {
		t.Fatalf("expected bundle path to round-trip, got %q", bundlePath)
	}
}
