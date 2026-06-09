package packaging

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"skill_registry/foundation/persist"
)

func TestPackagingWorkspaceRuntimePreparePersistsManagedWorkspaceAndReturnsHandle(t *testing.T) {
	root := t.TempDir()
	store := newMemoryWorkspaceStore()
	runtime := newPackagingWorkspaceRuntime(root, store)

	response, err := runtime.Prepare(context.Background(), PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   Target{PackageName: "@platform.skill/runtime-prepare", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "runtime-prepare", Description: "desc", SourceType: "native"},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}
	if response.WorkspacePath == "" {
		t.Fatal("expected workspace handle")
	}
	record, err := store.LoadWorkspace(context.Background(), response.WorkspacePath)
	if err != nil {
		t.Fatalf("expected persisted workspace row: %v", err)
	}
	if record.WorkspaceID != response.WorkspacePath {
		t.Fatalf("expected workspace handle %q, got record %#v", response.WorkspacePath, record)
	}
	expectedManagedWorkspace := filepath.Join(root, response.WorkspacePath, "workspace")
	if record.WorkspacePath != expectedManagedWorkspace {
		t.Fatalf("expected managed workspace path %q, got %q", expectedManagedWorkspace, record.WorkspacePath)
	}
}

func TestPackagingWorkspaceRuntimeValidateUsesCachedValidationResultFromWorkspaceStore(t *testing.T) {
	root := t.TempDir()
	store := newCountingWorkspaceStore()
	runtime := newPackagingWorkspaceRuntime(root, store)

	prepareResp, err := runtime.Prepare(context.Background(), PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   Target{PackageName: "@platform.skill/runtime-validate", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "runtime-validate", Description: "desc", SourceType: "native"},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}

	first, err := runtime.Validate(context.Background(), ValidateRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/runtime-validate", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("first validate returned error: %v", err)
	}
	record, err := store.LoadWorkspace(context.Background(), prepareResp.WorkspacePath)
	if err != nil {
		t.Fatalf("load workspace record after first validate: %v", err)
	}
	if record.ValidationCache == nil {
		t.Fatal("expected first validate to persist validation cache")
	}
	if err := os.WriteFile(filepath.Join(record.WorkspacePath, "package.json"), []byte("{"), 0o644); err != nil {
		t.Fatalf("corrupt package.json after first validate: %v", err)
	}

	second, err := runtime.Validate(context.Background(), ValidateRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/runtime-validate", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("second validate returned error: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expected cached validation response, got first=%#v second=%#v", first, second)
	}
	if store.loadWorkspaceCalls != 3 {
		t.Fatalf("expected first validate, record lookup, and second validate loads only, got %d", store.loadWorkspaceCalls)
	}
}

func TestPackagingWorkspaceRuntimeBundleContentRebuildsManagedBundleWhenStoredFileMissing(t *testing.T) {
	root := t.TempDir()
	store := newMemoryWorkspaceStore()
	runtime := newPackagingWorkspaceRuntime(root, store)

	prepareResp, err := runtime.Prepare(context.Background(), PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   Target{PackageName: "@platform.skill/runtime-bundle", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "runtime-bundle", Description: "desc", SourceType: "native"},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}

	if _, err := runtime.Pack(context.Background(), PackRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/runtime-bundle", Version: "1.0.0"},
	}); err != nil {
		t.Fatalf("pack returned error: %v", err)
	}

	record, err := store.LoadWorkspace(context.Background(), prepareResp.WorkspacePath)
	if err != nil {
		t.Fatalf("load workspace after pack: %v", err)
	}
	originalContent, err := os.ReadFile(record.BundlePath)
	if err != nil {
		t.Fatalf("read persisted bundle before deletion: %v", err)
	}
	if err := os.Remove(record.BundlePath); err != nil {
		t.Fatalf("remove persisted bundle: %v", err)
	}

	rebuiltContent, err := runtime.BundleContent(context.Background(), prepareResp.WorkspacePath, Target{
		PackageName: "@platform.skill/runtime-bundle",
		Version:     "1.0.0",
	})
	if err != nil {
		t.Fatalf("bundle content should rebuild missing bundle: %v", err)
	}
	if !bytes.Equal(originalContent, rebuiltContent) {
		t.Fatal("expected rebuilt bundle content to match original bundle content")
	}
	if _, err := os.Stat(record.BundlePath); err != nil {
		t.Fatalf("expected rebuilt bundle file to exist, got %v", err)
	}
}

func TestPackagingWorkspaceRuntimeCleanupRejectsNonCanonicalManagedWorkspacePath(t *testing.T) {
	root := t.TempDir()
	store := newMemoryWorkspaceStore()
	runtime := newPackagingWorkspaceRuntime(root, store)

	poisonRoot := filepath.Join(root, "ws-poison")
	poisonPath := filepath.Join(poisonRoot, "unexpected")
	if err := os.MkdirAll(poisonPath, 0o755); err != nil {
		t.Fatalf("mkdir poison path: %v", err)
	}
	store.records["ws-poison"] = persist.WorkspaceRecord{
		WorkspaceID:   "ws-poison",
		WorkspacePath: poisonPath,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	err := runtime.Cleanup(context.Background(), "ws-poison")
	if err == nil {
		t.Fatal("expected cleanup to reject non-canonical managed path")
	}
	if _, statErr := os.Stat(poisonRoot); statErr != nil {
		t.Fatalf("expected poison root to remain after failed cleanup, got %v", statErr)
	}
}

func TestPackagingWorkspaceRuntimePruneExpiredWorkspacesSkipsWorkspaceRefreshedAfterListing(t *testing.T) {
	root := t.TempDir()
	store := newMemoryWorkspaceStore()
	runtime := newPackagingWorkspaceRuntime(root, store)

	prepareResp, err := runtime.Prepare(context.Background(), PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   Target{PackageName: "@platform.skill/runtime-refreshed", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "runtime-refreshed", Description: "desc", SourceType: "native"},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}

	store.mu.Lock()
	record := store.records[prepareResp.WorkspacePath]
	record.UpdatedAt = time.Now().UTC().Add(-2 * time.Hour)
	store.records[prepareResp.WorkspacePath] = record
	store.afterListExpired = func() {
		store.mu.Lock()
		defer store.mu.Unlock()
		refreshed := store.records[prepareResp.WorkspacePath]
		refreshed.UpdatedAt = time.Now().UTC()
		store.records[prepareResp.WorkspacePath] = refreshed
	}
	store.mu.Unlock()

	if err := runtime.PruneExpiredWorkspaces(context.Background(), time.Hour); err != nil {
		t.Fatalf("prune refreshed managed workspace: %v", err)
	}

	managedRoot := filepath.Join(root, prepareResp.WorkspacePath)
	if _, err := os.Stat(managedRoot); err != nil {
		t.Fatalf("expected refreshed managed root to remain, got %v", err)
	}
	record, err = store.LoadWorkspace(context.Background(), prepareResp.WorkspacePath)
	if err != nil {
		t.Fatalf("expected refreshed store row to remain, got %v", err)
	}
	if !record.UpdatedAt.After(time.Now().UTC().Add(-time.Minute)) {
		t.Fatalf("expected refreshed updated_at to stay fresh, got %s", record.UpdatedAt)
	}
}
