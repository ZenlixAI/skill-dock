package packaging

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
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
