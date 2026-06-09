# Business Packaging Runtime / Published Release Deepening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deepen `business/packaging` by extracting a `Packaging Workspace Runtime` module and a `Published Release Workflow` module while keeping `packaging.Service` and all caller-facing contracts compatible.

**Architecture:** Keep `Service` as the compatibility facade, but move `Packaging Workspace` lifecycle, bundle access, cache, lock, and pruning behavior behind a dedicated runtime seam. Move publish recovery and `Release Record` orchestration behind a separate workflow seam that depends on the runtime instead of reimplementing pack and bundle access logic inside the facade.

**Tech Stack:** Go, `go test`, package-local module tests in `business/packaging`, compatibility verification through `app/catalogapi`

---

## File Map

**Create:**
- `business/packaging/workspace_runtime_module.go`
- `business/packaging/workspace_runtime_module_test.go`
- `business/packaging/published_release_workflow.go`
- `business/packaging/published_release_workflow_test.go`

**Modify:**
- `business/packaging/service.go`
- `business/packaging/service_test.go`

**Verify only:**
- `app/catalogapi/packaging_module.go`
- `app/catalogapi/packaging_publish.go`

## Task 1: Extract `Packaging Workspace Runtime` Prepare / Validate / Pack Seam

**Files:**
- Create: `business/packaging/workspace_runtime_module.go`
- Create: `business/packaging/workspace_runtime_module_test.go`
- Modify: `business/packaging/service.go`
- Test: `business/packaging/workspace_runtime_module_test.go`

- [ ] **Step 1: Write the failing runtime tests for managed prepare and cached validate**

```go
package packaging

import (
	"context"
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
	second, err := runtime.Validate(context.Background(), ValidateRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/runtime-validate", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("second validate returned error: %v", err)
	}
	if !reflect.DeepEqual(first.Components, second.Components) {
		t.Fatalf("expected cached validation response, got first=%#v second=%#v", first, second)
	}
	if store.loadWorkspaceCalls != 2 {
		t.Fatalf("expected workspace loads only, got %d", store.loadWorkspaceCalls)
	}
}
```

- [ ] **Step 2: Run the runtime tests to verify RED**

Run: `env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/packaging -run 'TestPackagingWorkspaceRuntime(PreparePersistsManagedWorkspaceAndReturnsHandle|ValidateUsesCachedValidationResultFromWorkspaceStore)'`

Expected: FAIL with compile errors such as `undefined: newPackagingWorkspaceRuntime` and missing `Prepare` / `Validate` methods on the new runtime module.

- [ ] **Step 3: Write the minimal runtime module and delegate `Service` prepare / validate / pack to it**

```go
package packaging

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"skill_registry/business/catalog"
	"skill_registry/foundation/persist"
)

type packagingWorkspaceRuntime struct {
	workspaceRoot   string
	workspaceSource workspaceSourceModule
	bundleRules     bundleRuleModule
	workspaceStore  persist.WorkspaceStore

	workspaceLocksMu sync.Mutex
	workspaceLocks   map[string]*workspaceLockEntry
}

func newPackagingWorkspaceRuntime(tempRoot string, store persist.WorkspaceStore) *packagingWorkspaceRuntime {
	if strings.TrimSpace(tempRoot) == "" {
		tempRoot = os.TempDir()
	}
	return &packagingWorkspaceRuntime{
		workspaceRoot:   tempRoot,
		workspaceSource: newWorkspaceSourceModule(tempRoot, 24*time.Hour),
		bundleRules:     newBundleRuleModule(),
		workspaceStore:  store,
		workspaceLocks:  map[string]*workspaceLockEntry{},
	}
}

func (r *packagingWorkspaceRuntime) Prepare(ctx context.Context, input PrepareRequest) (PrepareResponse, error) {
	if err := validateTarget(input.Target); err != nil {
		return PrepareResponse{}, err
	}
	if err := packagingContextError(ctx); err != nil {
		return PrepareResponse{}, err
	}

	workspacePath, err := r.workspaceSource.prepare(input)
	if err != nil {
		return PrepareResponse{}, err
	}
	outcome, err := r.bundleRules.prepareWorkspace(workspacePath, input)
	if err != nil {
		return PrepareResponse{}, err
	}
	if r.workspaceStore != nil {
		handle, _, persistErr := r.persistManagedWorkspace(ctx, workspacePath)
		if persistErr != nil {
			_ = os.RemoveAll(workspacePath)
			return PrepareResponse{}, persistErr
		}
		_ = os.RemoveAll(workspacePath)
		workspacePath = handle
	}
	return PrepareResponse{
		WorkspacePath:  workspacePath,
		PackageName:    input.Target.PackageName,
		Version:        input.Target.Version,
		Warnings:       outcome.Warnings,
		WarningDetails: warningDetails(outcome.Warnings),
		ImportReport:   outcome.ImportReport,
		Errors:         []string{},
	}, nil
}

func (r *packagingWorkspaceRuntime) Validate(ctx context.Context, input ValidateRequest) (ValidateResponse, error) {
	if err := validateTarget(input.Target); err != nil {
		return ValidateResponse{}, err
	}
	if err := packagingContextError(ctx); err != nil {
		return ValidateResponse{}, err
	}
	ctx, unlock, err := r.lockWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return ValidateResponse{}, err
	}
	defer unlock()

	return r.validateLocked(ctx, input)
}

func (r *packagingWorkspaceRuntime) Pack(ctx context.Context, input PackRequest) (PackResponse, error) {
	if err := validateTarget(input.Target); err != nil {
		return PackResponse{}, err
	}
	if err := packagingContextError(ctx); err != nil {
		return PackResponse{}, err
	}
	ctx, unlock, err := r.lockWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return PackResponse{}, err
	}
	defer unlock()

	result, err := r.packLocked(ctx, input)
	if err != nil {
		return PackResponse{}, err
	}
	defer result.cleanup()
	return result.response, nil
}
```

```go
type Service struct {
	workspaceRuntime *packagingWorkspaceRuntime
	publishWorkflow  *publishedReleaseWorkflow
}

func NewServiceWithWorkspaceStore(tempRoot string, store persist.WorkspaceStore) *Service {
	runtime := newPackagingWorkspaceRuntime(tempRoot, store)
	return &Service{
		workspaceRuntime: runtime,
		publishWorkflow:  newPublishedReleaseWorkflow(runtime, nil, nil),
	}
}

func (s *Service) PrepareContext(ctx context.Context, input PrepareRequest) (PrepareResponse, error) {
	return s.workspaceRuntime.Prepare(ctx, input)
}

func (s *Service) ValidateContext(ctx context.Context, input ValidateRequest) (ValidateResponse, error) {
	return s.workspaceRuntime.Validate(ctx, input)
}

func (s *Service) PackContext(ctx context.Context, input PackRequest) (PackResponse, error) {
	return s.workspaceRuntime.Pack(ctx, input)
}
```

```go
func (r *packagingWorkspaceRuntime) validateLocked(ctx context.Context, input ValidateRequest) (ValidateResponse, error) {
	if r.workspaceStore != nil {
		record, err := r.loadManagedWorkspaceRecord(ctx, input.WorkspacePath)
		if err != nil {
			return ValidateResponse{}, err
		}
		if cached, ok := cachedValidationResult(record.ValidationCache, input.Target); ok {
			return validateResponseFromResult(input, cached, record.WorkspacePath), nil
		}
		result, err := r.bundleRules.validate(record.WorkspacePath, input.Target)
		if err != nil {
			return ValidateResponse{}, err
		}
		if saveErr := r.workspaceStore.SaveValidationCache(ctx, input.WorkspacePath, workspaceValidationCacheFromResult(input.Target, result), time.Now().UTC()); saveErr != nil {
			return ValidateResponse{}, catalog.NewError(500, "internal_error", "Failed to persist packaging workspace validation cache.", nil)
		}
		return validateResponseFromResult(input, result, record.WorkspacePath), nil
	}

	workspacePath, cleanup, err := r.materializeWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return ValidateResponse{}, err
	}
	defer cleanup()

	result, err := r.bundleRules.validate(workspacePath, input.Target)
	if err != nil {
		return ValidateResponse{}, err
	}
	return validateResponseFromResult(input, result, workspacePath), nil
}

func (r *packagingWorkspaceRuntime) packLocked(ctx context.Context, input PackRequest) (packExecutionResult, error) {
	if r.workspaceStore != nil {
		record, err := r.loadManagedWorkspaceRecord(ctx, input.WorkspacePath)
		if err != nil {
			return packExecutionResult{}, err
		}
		result, ok := cachedValidationResult(record.ValidationCache, input.Target)
		if !ok {
			result, err = r.bundleRules.validate(record.WorkspacePath, input.Target)
			if err != nil {
				return packExecutionResult{}, err
			}
			if saveErr := r.workspaceStore.SaveValidationCache(ctx, input.WorkspacePath, workspaceValidationCacheFromResult(input.Target, result), time.Now().UTC()); saveErr != nil {
				return packExecutionResult{}, catalog.NewError(500, "internal_error", "Failed to persist packaging workspace validation cache.", nil)
			}
		}
		response := PackResponse{
			WorkspacePath:  input.WorkspacePath,
			PackageName:    input.Target.PackageName,
			Version:        input.Target.Version,
			ManifestDigest: result.ManifestDigest,
			Components:     append([]string(nil), result.Components...),
			Warnings:       append([]string(nil), result.Warnings...),
			WarningDetails: warningDetails(result.Warnings),
			ImportReport:   readImportReport(record.WorkspacePath, result.Warnings),
			Errors:         append([]string(nil), result.Errors...),
		}
		if len(result.Errors) > 0 {
			return packExecutionResult{response: response, materializedPath: record.WorkspacePath, cleanup: func() {}}, nil
		}
		bundlePath := r.managedBundlePath(input.WorkspacePath, input.Target)
		if bundleBase64, reused, err := reuseManagedBundle(record, input, bundlePath); err != nil {
			return packExecutionResult{}, err
		} else if reused {
			response.BundlePath = workspaceBundleHandle(input.WorkspacePath, input.Target)
			response.BundleBase64 = bundleBase64
			return packExecutionResult{response: response, materializedPath: record.WorkspacePath, bundlePath: bundlePath, cleanup: func() {}}, nil
		}
		bundleBase64, err := r.buildManagedBundle(record.WorkspacePath, input.WorkspacePath, input.Target, input.IncludeBundle)
		if err != nil {
			return packExecutionResult{}, err
		}
		if saveErr := r.workspaceStore.SaveBundle(ctx, input.WorkspacePath, bundlePath, time.Now().UTC()); saveErr != nil {
			return packExecutionResult{}, catalog.NewError(500, "internal_error", "Packed bundle could not be persisted.", nil)
		}
		response.BundlePath = workspaceBundleHandle(input.WorkspacePath, input.Target)
		response.BundleBase64 = bundleBase64
		return packExecutionResult{response: response, materializedPath: record.WorkspacePath, bundlePath: bundlePath, cleanup: func() {}}, nil
	}

	workspacePath, cleanup, err := r.materializeWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return packExecutionResult{}, err
	}
	result, err := r.bundleRules.validate(workspacePath, input.Target)
	if err != nil {
		cleanup()
		return packExecutionResult{}, err
	}
	response := PackResponse{
		WorkspacePath:  input.WorkspacePath,
		PackageName:    input.Target.PackageName,
		Version:        input.Target.Version,
		ManifestDigest: result.ManifestDigest,
		Components:     result.Components,
		Warnings:       result.Warnings,
		WarningDetails: warningDetails(result.Warnings),
		ImportReport:   readImportReport(workspacePath, result.Warnings),
		Errors:         result.Errors,
	}
	bundlePath, err := r.workspaceSource.bundlePath(workspacePath, input.Target)
	if err != nil {
		cleanup()
		return packExecutionResult{}, err
	}
	bundleBase64, err := r.bundleRules.buildBundle(workspacePath, bundlePath, input.IncludeBundle)
	if err != nil {
		cleanup()
		return packExecutionResult{}, err
	}
	response.BundlePath = bundlePath
	response.BundleBase64 = bundleBase64
	return packExecutionResult{response: response, materializedPath: workspacePath, bundlePath: bundlePath, cleanup: cleanup}, nil
}
```

- [ ] **Step 4: Run the runtime tests and the full packaging package to verify GREEN**

Run: `env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/packaging -run 'TestPackagingWorkspaceRuntime(PreparePersistsManagedWorkspaceAndReturnsHandle|ValidateUsesCachedValidationResultFromWorkspaceStore)'`

Expected: PASS

Run: `env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/packaging`

Expected: PASS with the existing `Service` tests still green.

- [ ] **Step 5: Commit the runtime extraction**

```bash
git add business/packaging/service.go business/packaging/workspace_runtime_module.go business/packaging/workspace_runtime_module_test.go
git commit -m "refactor: extract packaging workspace runtime"
```

## Task 2: Move Bundle Access, Metadata, Cleanup, and Pruning into the Runtime

**Files:**
- Modify: `business/packaging/workspace_runtime_module.go`
- Modify: `business/packaging/workspace_runtime_module_test.go`
- Modify: `business/packaging/service.go`
- Modify: `business/packaging/service_test.go`
- Test: `business/packaging/workspace_runtime_module_test.go`

- [ ] **Step 1: Write the failing runtime tests for bundle reload, cleanup safety, and pruning**

```go
func TestPackagingWorkspaceRuntimeBundleContentRepacksManagedBundleWhenStoredBundleIsMissing(t *testing.T) {
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

	_, err = runtime.Pack(context.Background(), PackRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/runtime-bundle", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("pack returned error: %v", err)
	}

	record, err := store.LoadWorkspace(context.Background(), prepareResp.WorkspacePath)
	if err != nil {
		t.Fatalf("load workspace record: %v", err)
	}
	if err := os.Remove(record.BundlePath); err != nil {
		t.Fatalf("remove bundle: %v", err)
	}

	content, err := runtime.BundleContent(context.Background(), prepareResp.WorkspacePath, Target{
		PackageName: "@platform.skill/runtime-bundle",
		Version:     "1.0.0",
	})
	if err != nil {
		t.Fatalf("bundle content returned error: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("expected runtime to rebuild missing managed bundle")
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
		Target:   Target{PackageName: "@platform.skill/runtime-prune", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "runtime-prune", Description: "desc", SourceType: "native"},
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

	if err := runtime.PruneExpiredWorkspaces(time.Hour); err != nil {
		t.Fatalf("prune returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, prepareResp.WorkspacePath)); err != nil {
		t.Fatalf("expected refreshed workspace to remain, got %v", err)
	}
}
```

- [ ] **Step 2: Run the runtime tests to verify RED**

Run: `env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/packaging -run 'TestPackagingWorkspaceRuntime(BundleContentRepacksManagedBundleWhenStoredBundleIsMissing|CleanupRejectsNonCanonicalManagedWorkspacePath|PruneExpiredWorkspacesSkipsWorkspaceRefreshedAfterListing)'`

Expected: FAIL because `BundleContent`, `Cleanup`, and `PruneExpiredWorkspaces` do not exist yet on the runtime module.

- [ ] **Step 3: Move the remaining workspace lifecycle methods to the runtime and delegate the facade**

```go
func (r *packagingWorkspaceRuntime) Cleanup(ctx context.Context, workspacePath string) error {
	if err := packagingContextError(ctx); err != nil {
		return err
	}
	ctx, unlock, err := r.lockWorkspace(ctx, workspacePath)
	if err != nil {
		return err
	}
	defer unlock()

	if r.workspaceStore != nil {
		record, err := r.workspaceStore.LoadWorkspace(ctx, workspacePath)
		if err != nil {
			if err == persist.ErrWorkspaceRecordNotFound {
				return catalog.NewError(422, "invalid_request", "workspacePath must reference an existing packaging workspace.", map[string]any{"workspacePath": workspacePath})
			}
			return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
		}
		managedRoot, err := r.managedWorkspaceRootFromPath(workspacePath, record.WorkspacePath)
		if err != nil {
			return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
		}
		if err := os.RemoveAll(managedRoot); err != nil {
			return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
		}
		if err := r.workspaceStore.DeleteWorkspace(ctx, workspacePath); err != nil {
			return catalog.NewError(500, "internal_error", "Failed to remove packaging workspace.", nil)
		}
		return nil
	}
	return r.workspaceSource.cleanup(workspacePath)
}

func (r *packagingWorkspaceRuntime) BundlePath(ctx context.Context, workspacePath string, target Target) (string, error) {
	if err := validateTarget(target); err != nil {
		return "", err
	}
	if err := packagingContextError(ctx); err != nil {
		return "", err
	}
	ctx, unlock, err := r.lockWorkspace(ctx, workspacePath)
	if err != nil {
		return "", err
	}
	defer unlock()

	if r.workspaceStore != nil {
		bundlePath, err := r.workspaceStore.LoadBundle(ctx, workspacePath)
		if err != nil {
			if err == persist.ErrWorkspaceRecordNotFound {
				return "", catalog.NewError(422, "invalid_request", "workspacePath must reference an existing packaging workspace.", map[string]any{"workspacePath": workspacePath})
			}
			if err == persist.ErrWorkspaceBundleNotFound {
				return "", catalog.NewError(404, "bundle_not_found", "Bundle archive does not exist for the requested workspace.", map[string]any{"workspacePath": workspacePath})
			}
			return "", catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
		}
		if err := r.validateManagedBundlePath(workspacePath, target, bundlePath); err != nil {
			return "", catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
		}
		return workspaceBundleHandle(workspacePath, target), nil
	}
	return r.workspaceSource.bundlePath(workspacePath, target)
}

func (r *packagingWorkspaceRuntime) BundleContent(ctx context.Context, workspacePath string, target Target) ([]byte, error) {
	if err := validateTarget(target); err != nil {
		return nil, err
	}
	if err := packagingContextError(ctx); err != nil {
		return nil, err
	}
	ctx, unlock, err := r.lockWorkspace(ctx, workspacePath)
	if err != nil {
		return nil, err
	}
	defer unlock()

	if r.workspaceStore != nil {
		bundlePath, err := r.workspaceStore.LoadBundle(ctx, workspacePath)
		if err == nil {
			if err := r.validateManagedBundlePath(workspacePath, target, bundlePath); err != nil {
				return nil, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
			}
			if content, readErr := os.ReadFile(bundlePath); readErr == nil {
				return content, nil
			}
		}
		packed, packErr := r.packForPublish(ctx, PackRequest{WorkspacePath: workspacePath, Target: target})
		if packErr != nil {
			return nil, packErr
		}
		defer packed.cleanup()
		return os.ReadFile(packed.bundlePath)
	}

	bundlePath, err := r.workspaceSource.bundlePath(workspacePath, target)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(bundlePath)
}

func (r *packagingWorkspaceRuntime) PackageMetadata(ctx context.Context, workspacePath string) (map[string]any, error) {
	ctx, unlock, err := r.lockWorkspace(ctx, workspacePath)
	if err != nil {
		return nil, err
	}
	defer unlock()

	if r.workspaceStore != nil {
		record, err := r.loadManagedWorkspaceRecord(ctx, workspacePath)
		if err != nil {
			return nil, err
		}
		if len(record.PackageMetadataJSON) > 0 {
			return decodeWorkspacePackageMetadataJSON(workspacePath, record.PackageMetadataJSON)
		}
		content, err := os.ReadFile(filepath.Join(record.WorkspacePath, "package.json"))
		if err != nil {
			return nil, catalog.NewError(422, "invalid_request", "workspace package.json could not be read.", map[string]any{"workspacePath": workspacePath})
		}
		payload, err := decodeWorkspacePackageMetadataJSON(workspacePath, content)
		if err != nil {
			return nil, err
		}
		if saveErr := r.workspaceStore.SavePackageMetadata(ctx, workspacePath, content, time.Now().UTC()); saveErr != nil {
			return nil, catalog.NewError(500, "internal_error", "Failed to persist packaging workspace package metadata cache.", nil)
		}
		return payload, nil
	}

	materializedPath, cleanup, err := r.materializeWorkspace(ctx, workspacePath)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	content, err := os.ReadFile(filepath.Join(materializedPath, "package.json"))
	if err != nil {
		return nil, catalog.NewError(422, "invalid_request", "workspace package.json could not be read.", map[string]any{"workspacePath": workspacePath})
	}
	return decodeWorkspacePackageMetadataJSON(workspacePath, content)
}

func (r *packagingWorkspaceRuntime) PruneExpiredWorkspaces(ttl time.Duration) error {
	if r.workspaceStore != nil && ttl > 0 {
		cutoff := time.Now().UTC().Add(-ttl)
		expiredRecords, err := r.workspaceStore.ListExpired(context.Background(), cutoff)
		if err != nil {
			return err
		}
		for _, record := range expiredRecords {
			lockCtx, unlock, lockErr := r.lockWorkspace(context.Background(), record.WorkspaceID)
			if lockErr != nil {
				return lockErr
			}
			current, loadErr := r.workspaceStore.LoadWorkspace(lockCtx, record.WorkspaceID)
			if loadErr == nil && current.UpdatedAt.Before(cutoff) {
				managedRoot, rootErr := r.managedWorkspaceRootFromPath(current.WorkspaceID, current.WorkspacePath)
				if rootErr != nil {
					unlock()
					return rootErr
				}
				if err := os.RemoveAll(managedRoot); err != nil {
					unlock()
					return err
				}
				if err := r.workspaceStore.DeleteWorkspace(lockCtx, record.WorkspaceID); err != nil {
					unlock()
					return err
				}
			} else if loadErr != nil && loadErr != persist.ErrWorkspaceRecordNotFound {
				unlock()
				return loadErr
			}
			unlock()
		}
	}
	return r.workspaceSource.pruneExpiredWorkspaces(ttl)
}
```

```go
func (s *Service) CleanupContext(ctx context.Context, workspacePath string) error {
	return s.workspaceRuntime.Cleanup(ctx, workspacePath)
}

func (s *Service) BundlePathContext(ctx context.Context, workspacePath string, target Target) (string, error) {
	return s.workspaceRuntime.BundlePath(ctx, workspacePath, target)
}

func (s *Service) BundleContentContext(ctx context.Context, workspacePath string, target Target) ([]byte, error) {
	return s.workspaceRuntime.BundleContent(ctx, workspacePath, target)
}

func (s *Service) PackageMetadataContext(ctx context.Context, workspacePath string) (map[string]any, error) {
	return s.workspaceRuntime.PackageMetadata(ctx, workspacePath)
}

func (s *Service) PruneExpiredWorkspaces(ttl time.Duration) error {
	return s.workspaceRuntime.PruneExpiredWorkspaces(ttl)
}
```

- [ ] **Step 4: Run the targeted runtime tests and legacy cleanup / prune coverage to verify GREEN**

Run: `env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/packaging -run 'TestPackagingWorkspaceRuntime(BundleContentRepacksManagedBundleWhenStoredBundleIsMissing|CleanupRejectsNonCanonicalManagedWorkspacePath|PruneExpiredWorkspacesSkipsWorkspaceRefreshedAfterListing)|Test(PruneExpiredWorkspacesWithWorkspaceStoreRemovesManagedWorkspaceTree|CleanupWithWorkspaceStoreRejectsNonCanonicalManagedWorkspacePath)'`

Expected: PASS

Run: `env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/packaging`

Expected: PASS

- [ ] **Step 5: Commit the runtime lifecycle move**

```bash
git add business/packaging/service.go business/packaging/service_test.go business/packaging/workspace_runtime_module.go business/packaging/workspace_runtime_module_test.go
git commit -m "refactor: move workspace lifecycle behind runtime"
```

## Task 3: Extract `Published Release Workflow`

**Files:**
- Create: `business/packaging/published_release_workflow.go`
- Create: `business/packaging/published_release_workflow_test.go`
- Modify: `business/packaging/service.go`
- Modify: `business/packaging/service_test.go`
- Test: `business/packaging/published_release_workflow_test.go`

- [ ] **Step 1: Write the failing publish workflow tests for release replay and publish recovery**

```go
package packaging

import (
	"context"
	"testing"

	"skill_registry/business/catalog"
)

func TestPublishedReleaseWorkflowPublishesAfterRegistryVersionNotFound(t *testing.T) {
	runtime := newPackagingWorkspaceRuntime(t.TempDir(), nil)
	repo := &stubReleaseRecordCreator{
		results: []stubReleaseRecordOutcome{
			{err: catalog.NewError(404, "registry_version_not_found", "Registry version not found.", nil)},
			{result: catalog.ReleaseRecordResult{Created: true, VersionStatus: catalog.VersionStatusPublished, InstallPolicy: catalog.InstallPolicyResolvable}},
		},
	}
	publisher := &capturePackagePublisher{}
	workflow := newPublishedReleaseWorkflow(runtime, repo, publisher)

	sourceDir := copyExternalSkillFixture(t, "minimal-imported-skill")
	prepareResp, err := runtime.Prepare(context.Background(), PrepareRequest{
		Source:   Source{Type: "upload_archive", ArchiveBase64: buildSourceArchiveBase64FromDirectory(t, sourceDir)},
		Target:   Target{PackageName: "@platform.skill/workflow-publish", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "workflow-publish", Description: "desc", SourceType: "imported"},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}

	response, err := workflow.Publish(context.Background(), PublishRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/workflow-publish", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if !response.PublishedToRegistry || !response.Created {
		t.Fatalf("unexpected publish response: %+v", response)
	}
	if repo.calls != 2 || publisher.calls != 1 {
		t.Fatalf("expected release retry and one package publish, repo=%d publisher=%d", repo.calls, publisher.calls)
	}
}

func TestPublishedReleaseWorkflowSkipsRegistryPublishWhenReleaseAlreadyExists(t *testing.T) {
	runtime := newPackagingWorkspaceRuntime(t.TempDir(), nil)
	repo := &stubReleaseRecordCreator{
		results: []stubReleaseRecordOutcome{
			{result: catalog.ReleaseRecordResult{Created: false, IdempotentReplay: true, VersionStatus: catalog.VersionStatusPublished, InstallPolicy: catalog.InstallPolicyResolvable}},
		},
	}
	publisher := &capturePackagePublisher{}
	workflow := newPublishedReleaseWorkflow(runtime, repo, publisher)

	sourceDir := copyExternalSkillFixture(t, "minimal-imported-skill")
	prepareResp, err := runtime.Prepare(context.Background(), PrepareRequest{
		Source:   Source{Type: "upload_archive", ArchiveBase64: buildSourceArchiveBase64FromDirectory(t, sourceDir)},
		Target:   Target{PackageName: "@platform.skill/workflow-replay", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "workflow-replay", Description: "desc", SourceType: "imported"},
	})
	if err != nil {
		t.Fatalf("prepare returned error: %v", err)
	}

	response, err := workflow.Publish(context.Background(), PublishRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/workflow-replay", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if response.PublishedToRegistry || response.Created || !response.IdempotentReplay {
		t.Fatalf("unexpected publish response: %+v", response)
	}
	if publisher.calls != 0 {
		t.Fatalf("expected publisher to be skipped, got %d", publisher.calls)
	}
}
```

- [ ] **Step 2: Run the publish workflow tests to verify RED**

Run: `env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/packaging -run 'TestPublishedReleaseWorkflow(PublishesAfterRegistryVersionNotFound|SkipsRegistryPublishWhenReleaseAlreadyExists)'`

Expected: FAIL with `undefined: newPublishedReleaseWorkflow` and missing `Publish` method on the workflow module.

- [ ] **Step 3: Implement the workflow and rewire `Service.ConfigurePublishWorkflow` / `PublishContext`**

```go
package packaging

import (
	"context"
	"os"

	"skill_registry/business/catalog"
	artifactadapter "skill_registry/foundation/artifact"
)

type publishedReleaseWorkflow struct {
	runtime        *packagingWorkspaceRuntime
	releaseRecords releaseRecordCreator
	publisher      packagePublisher
}

func newPublishedReleaseWorkflow(runtime *packagingWorkspaceRuntime, repo releaseRecordCreator, publisher packagePublisher) *publishedReleaseWorkflow {
	return &publishedReleaseWorkflow{
		runtime:        runtime,
		releaseRecords: repo,
		publisher:      publisher,
	}
}

func (w *publishedReleaseWorkflow) Configure(repo releaseRecordCreator, publisher packagePublisher) {
	w.releaseRecords = repo
	w.publisher = publisher
}

func (w *publishedReleaseWorkflow) Publish(ctx context.Context, input PublishRequest) (PublishResponse, error) {
	if err := packagingContextError(ctx); err != nil {
		return PublishResponse{}, err
	}
	if w.releaseRecords == nil || w.publisher == nil {
		return PublishResponse{}, catalog.NewError(500, "internal_error", "Publish workflow is not configured.", nil)
	}

	packResult, err := w.runtime.packForPublish(ctx, PackRequest{
		WorkspacePath: input.WorkspacePath,
		Target:        input.Target,
	})
	if err != nil {
		return PublishResponse{}, err
	}
	defer packResult.cleanup()

	packResponse := packResult.response
	if len(packResponse.Errors) > 0 {
		return PublishResponse{}, catalog.NewError(422, "bundle_validation_failed", "Bundle validation failed before publish.", map[string]any{
			"packageName": input.Target.PackageName,
			"version":     input.Target.Version,
			"errors":      packResponse.Errors,
		})
	}

	releaseInput := catalog.ReleaseRecordInput{
		ManifestDigest: stringPtr(packResponse.ManifestDigest),
		Components:     packResponse.Components,
	}
	releaseResult, err := w.releaseRecords.CreateReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
	if err == nil {
		return publishResponseFromPack(packResponse, releaseResult, false), nil
	}

	releaseErr := packagingCatalogError(err)
	if releaseErr.Code != "registry_version_not_found" {
		return PublishResponse{}, releaseErr
	}

	packageMetadata, err := readWorkspacePackageMetadataFile(input.WorkspacePath, packResult.materializedPath)
	if err != nil {
		return PublishResponse{}, err
	}
	tarball, err := os.ReadFile(packResult.bundlePath)
	if err != nil {
		return PublishResponse{}, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil)
	}

	publishErr := w.publisher.PublishPackage(ctx, artifactadapter.NPMPublishInput{
		PackageName: input.Target.PackageName,
		Version:     input.Target.Version,
		Description: packageMetadata.Description,
		DisplayName: packageMetadata.Skill.DisplayName,
		SourceType:  packageMetadata.Skill.SourceType,
		Tarball:     tarball,
	})
	if publishErr != nil {
		catalogErr := packagingCatalogError(publishErr)
		if catalogErr.Code == "registry_version_conflict" {
			releaseResult, err = w.releaseRecords.CreateReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
			if err == nil {
				return publishResponseFromPack(packResponse, releaseResult, false), nil
			}
		}
		return PublishResponse{}, catalogErr
	}

	releaseResult, err = w.releaseRecords.CreateReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
	if err != nil {
		return PublishResponse{}, packagingCatalogError(err)
	}
	return publishResponseFromPack(packResponse, releaseResult, true), nil
}
```

```go
func (s *Service) ConfigurePublishWorkflow(repo releaseRecordCreator, publisher packagePublisher) {
	s.publishWorkflow.Configure(repo, publisher)
}

func (s *Service) PublishContext(ctx context.Context, input PublishRequest) (PublishResponse, error) {
	return s.publishWorkflow.Publish(ctx, input)
}
```

```go
func (r *packagingWorkspaceRuntime) packForPublish(ctx context.Context, input PackRequest) (packExecutionResult, error) {
	if err := validateTarget(input.Target); err != nil {
		return packExecutionResult{}, err
	}
	if err := packagingContextError(ctx); err != nil {
		return packExecutionResult{}, err
	}
	ctx, unlock, err := r.lockWorkspace(ctx, input.WorkspacePath)
	if err != nil {
		return packExecutionResult{}, err
	}
	defer unlock()

	return r.packLocked(ctx, input)
}
```

- [ ] **Step 4: Run the workflow tests and the legacy service publish tests to verify GREEN**

Run: `env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/packaging -run 'TestPublishedReleaseWorkflow(PublishesAfterRegistryVersionNotFound|SkipsRegistryPublishWhenReleaseAlreadyExists)|TestPublishContext(PublishesAndCreatesReleaseRecord|SkipsRegistryPublishWhenReleaseAlreadyExists)'`

Expected: PASS

Run: `env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/packaging`

Expected: PASS

- [ ] **Step 5: Commit the publish workflow extraction**

```bash
git add business/packaging/service.go business/packaging/service_test.go business/packaging/published_release_workflow.go business/packaging/published_release_workflow_test.go business/packaging/workspace_runtime_module.go
git commit -m "refactor: extract published release workflow"
```

## Task 4: Slim the Facade and Re-Anchor Compatibility Coverage

**Files:**
- Modify: `business/packaging/service.go`
- Modify: `business/packaging/service_test.go`
- Modify: `business/packaging/workspace_runtime_module_test.go`
- Modify: `business/packaging/published_release_workflow_test.go`
- Test: `business/packaging/service_test.go`

- [ ] **Step 1: Write the failing compatibility tests that prove the facade delegates without losing caller behavior**

```go
func TestServiceFacadeWithWorkspaceStoreBlocksCleanupUntilPublishCompletes(t *testing.T) {
	root := t.TempDir()
	store := newMemoryWorkspaceStore()
	service := NewServiceWithWorkspaceStore(root, store)
	repo := &stubReleaseRecordCreator{
		results: []stubReleaseRecordOutcome{
			{err: catalog.NewError(404, "registry_version_not_found", "Registry version not found.", nil)},
			{result: catalog.ReleaseRecordResult{Created: true, VersionStatus: catalog.VersionStatusPublished, InstallPolicy: catalog.InstallPolicyResolvable}},
		},
	}
	publisher := &capturePackagePublisher{
		started: make(chan struct{}),
		block:   make(chan struct{}),
	}
	service.ConfigurePublishWorkflow(repo, publisher)

	prepareResp, err := service.Prepare(PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   Target{PackageName: "@platform.skill/facade-managed", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "facade-managed", Description: "desc", SourceType: "native"},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	publishDone := make(chan error, 1)
	go func() {
		_, publishErr := service.PublishContext(context.Background(), PublishRequest{
			WorkspacePath: prepareResp.WorkspacePath,
			Target:        Target{PackageName: "@platform.skill/facade-managed", Version: "1.0.0"},
		})
		publishDone <- publishErr
	}()

	<-publisher.started

	cleanupDone := make(chan error, 1)
	go func() {
		cleanupDone <- service.Cleanup(prepareResp.WorkspacePath)
	}()

	select {
	case err := <-cleanupDone:
		t.Fatalf("expected cleanup to wait for publish completion, got %v", err)
	case <-time.After(200 * time.Millisecond):
	}

	close(publisher.block)
	if err := <-publishDone; err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	if err := <-cleanupDone; err != nil {
		t.Fatalf("cleanup returned error after publish: %v", err)
	}
}

func TestServiceFacadePublishAvoidsRedundantWorkspaceAndBundleLoads(t *testing.T) {
	root := t.TempDir()
	store := newCountingWorkspaceStore()
	service := NewServiceWithWorkspaceStore(root, store)
	repo := &stubReleaseRecordCreator{
		results: []stubReleaseRecordOutcome{
			{err: catalog.NewError(404, "registry_version_not_found", "Registry version not found.", nil)},
			{result: catalog.ReleaseRecordResult{Created: true, VersionStatus: catalog.VersionStatusPublished, InstallPolicy: catalog.InstallPolicyResolvable}},
		},
	}
	publisher := &capturePackagePublisher{}
	service.ConfigurePublishWorkflow(repo, publisher)

	prepareResp, err := service.Prepare(PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"docs/README.md": "# Capability\n",
			},
		},
		Target:   Target{PackageName: "@platform.skill/facade-count", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "facade-count", Description: "desc", SourceType: "native"},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	_, err = service.PublishContext(context.Background(), PublishRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/facade-count", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if store.loadWorkspaceCalls != 1 {
		t.Fatalf("expected one workspace load during publish, got %d", store.loadWorkspaceCalls)
	}
	if store.loadBundleCalls != 0 {
		t.Fatalf("expected zero bundle store reloads during publish, got %d", store.loadBundleCalls)
	}
}
```

- [ ] **Step 2: Run the compatibility tests to verify RED**

Run: `env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/packaging -run 'TestServiceFacade(WithWorkspaceStoreBlocksCleanupUntilPublishCompletes|PublishAvoidsRedundantWorkspaceAndBundleLoads)'`

Expected: FAIL if the facade still owns stale fields or the runtime / workflow split changed lock behavior.

- [ ] **Step 3: Remove stale facade fields and keep only wiring / compatibility methods**

```go
type Service struct {
	workspaceRuntime *packagingWorkspaceRuntime
	publishWorkflow  *publishedReleaseWorkflow
}

func NewService(tempRoot string) *Service {
	return NewServiceWithWorkspaceStore(tempRoot, nil)
}

func NewServiceWithWorkspaceStore(tempRoot string, store persist.WorkspaceStore) *Service {
	runtime := newPackagingWorkspaceRuntime(tempRoot, store)
	workflow := newPublishedReleaseWorkflow(runtime, nil, nil)
	return &Service{
		workspaceRuntime: runtime,
		publishWorkflow:  workflow,
	}
}

func (s *Service) Prepare(input PrepareRequest) (PrepareResponse, error) {
	return s.PrepareContext(context.Background(), input)
}

func (s *Service) Validate(input ValidateRequest) (ValidateResponse, error) {
	return s.ValidateContext(context.Background(), input)
}

func (s *Service) Pack(input PackRequest) (PackResponse, error) {
	return s.PackContext(context.Background(), input)
}

func (s *Service) Cleanup(workspacePath string) error {
	return s.CleanupContext(context.Background(), workspacePath)
}

func (s *Service) BundlePath(workspacePath string, target Target) (string, error) {
	return s.BundlePathContext(context.Background(), workspacePath, target)
}

func (s *Service) BundleContent(workspacePath string, target Target) ([]byte, error) {
	return s.BundleContentContext(context.Background(), workspacePath, target)
}

func (s *Service) PackageMetadata(workspacePath string) (map[string]any, error) {
	return s.PackageMetadataContext(context.Background(), workspacePath)
}

func (s *Service) Publish(input PublishRequest) (PublishResponse, error) {
	return s.PublishContext(context.Background(), input)
}
```

- [ ] **Step 4: Run the complete verification suite**

Run: `env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/packaging`

Expected: PASS

Run: `env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./app/catalogapi`

Expected: PASS, proving `packaging.Service` compatibility still satisfies upper-layer callers.

- [ ] **Step 5: Commit the facade slimming**

```bash
git add business/packaging/service.go business/packaging/service_test.go business/packaging/workspace_runtime_module_test.go business/packaging/published_release_workflow_test.go
git commit -m "refactor: slim packaging service facade"
```

## Self-Review

- Spec coverage:
  - `Packaging Workspace Runtime` extraction is covered by Task 1 and Task 2.
  - `Published Release Workflow` extraction is covered by Task 3.
  - `Service` compatibility facade is preserved and re-verified in Task 4.
  - `business/packaging` package verification and `app/catalogapi` compatibility verification are explicitly included.
- Placeholder scan:
  - No `TODO`, `TBD`, or “handle later” language remains.
  - Every task includes exact files, test names, commands, and commit messages.
- Type consistency:
  - Runtime method names stay consistent: `Prepare`, `Validate`, `Pack`, `Cleanup`, `BundlePath`, `BundleContent`, `PackageMetadata`, `PruneExpiredWorkspaces`.
  - Publish workflow method name stays consistent: `Publish`.
  - `Service` delegates to the same method names throughout the plan.
