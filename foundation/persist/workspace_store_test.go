package persist

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
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

func TestSQLWorkspaceStoreRoundTripsManagedWorkspaceCache(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "workspace-store-cache.db"))
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
	validationAt := createdAt.Add(5 * time.Minute)
	metadataAt := createdAt.Add(10 * time.Minute)

	if err := store.SaveWorkspace(ctx, "ws-123", "/tmp/workspaces/ws-123", createdAt); err != nil {
		t.Fatalf("save workspace: %v", err)
	}

	validationCache := WorkspaceValidationCache{
		PackageName:    "@platform.skill/zoom-out",
		Version:        "1.0.0",
		ManifestDigest: "sha256:abc123",
		Components:     []string{"docs"},
		Warnings:       []string{"normalized_missing_package_json"},
	}
	if err := store.SaveValidationCache(ctx, "ws-123", validationCache, validationAt); err != nil {
		t.Fatalf("save validation cache: %v", err)
	}

	packageMetadata := []byte(`{"name":"@platform.skill/zoom-out","version":"1.0.0","description":"Broader context","skill":{"displayName":"Zoom Out","sourceType":"imported"}}`)
	if err := store.SavePackageMetadata(ctx, "ws-123", packageMetadata, metadataAt); err != nil {
		t.Fatalf("save package metadata: %v", err)
	}

	record, err := store.LoadWorkspace(ctx, "ws-123")
	if err != nil {
		t.Fatalf("load workspace: %v", err)
	}
	if record.ValidationCache == nil {
		t.Fatal("expected validation cache to round-trip")
	}
	if record.ValidationCache.PackageName != validationCache.PackageName {
		t.Fatalf("expected cached package name %q, got %q", validationCache.PackageName, record.ValidationCache.PackageName)
	}
	if record.ValidationCache.Version != validationCache.Version {
		t.Fatalf("expected cached version %q, got %q", validationCache.Version, record.ValidationCache.Version)
	}
	if record.ValidationCache.ManifestDigest != validationCache.ManifestDigest {
		t.Fatalf("expected cached manifest digest %q, got %q", validationCache.ManifestDigest, record.ValidationCache.ManifestDigest)
	}
	if !reflect.DeepEqual(record.ValidationCache.Components, validationCache.Components) {
		t.Fatalf("expected cached components %#v, got %#v", validationCache.Components, record.ValidationCache.Components)
	}
	if !reflect.DeepEqual(record.ValidationCache.Warnings, validationCache.Warnings) {
		t.Fatalf("expected cached warnings %#v, got %#v", validationCache.Warnings, record.ValidationCache.Warnings)
	}
	if string(record.PackageMetadataJSON) != string(packageMetadata) {
		t.Fatalf("expected cached package metadata %s, got %s", string(packageMetadata), string(record.PackageMetadataJSON))
	}
	if !record.UpdatedAt.Equal(metadataAt) {
		t.Fatalf("expected updated_at %s after package metadata save, got %s", metadataAt, record.UpdatedAt)
	}
}

func TestSQLWorkspaceStoreMigratesLegacySchemaToManagedPaths(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "workspace-store-legacy.db"))
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = db.ExecContext(ctx, `
CREATE TABLE packaging_workspaces (
	workspace_id TEXT NOT NULL PRIMARY KEY,
	workspace_snapshot BLOB NOT NULL,
	bundle_content BLOB,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
)`)
	if err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO packaging_workspaces (workspace_id, workspace_snapshot, bundle_content, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)`,
		"legacy-ws",
		[]byte("snapshot"),
		[]byte("bundle"),
		"2026-06-08T10:00:00Z",
		"2026-06-08T10:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	store := newSQLWorkspaceStore(db, "sqlite")
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate legacy workspace store: %v", err)
	}

	legacyRecord, err := store.LoadWorkspace(ctx, "legacy-ws")
	if !errors.Is(err, ErrWorkspaceRecordNotFound) {
		t.Fatalf("expected legacy row without workspace_path to be treated as missing, got record=%#v err=%v", legacyRecord, err)
	}

	createdAt := time.Date(2026, time.June, 8, 11, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)
	if err := store.SaveWorkspace(ctx, "legacy-ws", "/managed/root/legacy-ws/workspace", createdAt); err != nil {
		t.Fatalf("save migrated workspace: %v", err)
	}
	if err := store.SaveBundle(ctx, "legacy-ws", "/managed/root/legacy-ws/.dist/bundle.tgz", updatedAt); err != nil {
		t.Fatalf("save migrated bundle: %v", err)
	}

	record, err := store.LoadWorkspace(ctx, "legacy-ws")
	if err != nil {
		t.Fatalf("load migrated workspace: %v", err)
	}
	if record.WorkspacePath != "/managed/root/legacy-ws/workspace" {
		t.Fatalf("expected migrated workspace path, got %q", record.WorkspacePath)
	}
	if record.BundlePath != "/managed/root/legacy-ws/.dist/bundle.tgz" {
		t.Fatalf("expected migrated bundle path, got %q", record.BundlePath)
	}
}

func TestSQLWorkspaceStoreListsExpiredManagedWorkspaces(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "workspace-store-expired.db"))
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

	expiredAt := time.Date(2026, time.June, 8, 9, 0, 0, 0, time.UTC)
	freshAt := expiredAt.Add(2 * time.Hour)
	if err := store.SaveWorkspace(ctx, "expired-ws", "/managed/root/expired-ws/workspace", expiredAt); err != nil {
		t.Fatalf("save expired workspace: %v", err)
	}
	if err := store.SaveWorkspace(ctx, "fresh-ws", "/managed/root/fresh-ws/workspace", freshAt); err != nil {
		t.Fatalf("save fresh workspace: %v", err)
	}

	records, err := store.ListExpired(ctx, expiredAt.Add(time.Hour))
	if err != nil {
		t.Fatalf("list expired workspaces: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 expired workspace, got %#v", records)
	}
	if records[0].WorkspaceID != "expired-ws" {
		t.Fatalf("expected expired workspace ID, got %#v", records[0])
	}
}
