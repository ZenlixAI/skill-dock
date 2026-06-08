# Packaging Shared Workspace Persistence Optimization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace snapshot/blob-based shared workspace persistence with managed shared workspace directories plus lightweight metadata so validate/pack/metadata reuse one persisted workspace and one persisted bundle path.

**Architecture:** Keep `workspaceId` as the external opaque handle, but change the internal persistence seam so `WorkspaceStore` records `workspace_path` and `bundle_path` metadata instead of storing whole workspace snapshots and bundle bytes. `packaging.Service` will move prepared workspaces into a configured shared root, operate directly on that managed workspace for validate/pack/metadata, and clean up the directory tree on delete.

**Tech Stack:** Go, SQLite/Postgres via `database/sql`, existing `business/packaging` service, existing `foundation/persist` SQL store tests

---

## File Map

- Create: `docs/superpowers/specs/2026-06-08-packaging-shared-workspace-persistence-optimization-design.zh-CN.md`
  - Approved design spec for the optimization
- Modify: `foundation/persist/workspace_store.go`
  - Convert the workspace store from blob fields to managed-path metadata fields
- Modify: `foundation/persist/workspace_store_test.go`
  - Cover the new SQL schema and store behavior
- Modify: `business/packaging/workspace_persistence.go`
  - Replace archive/materialize helpers with managed workspace path helpers
- Modify: `business/packaging/service.go`
  - Rewire prepare/validate/pack/bundle/cleanup to use managed workspace directories
- Modify: `business/packaging/service_test.go`
  - Add regression coverage for reuse of managed workspaces and persisted bundle paths
- Modify: `cmd/skill-registry/main.go`
  - Inject `SKILL_REGISTRY_WORKSPACE_ROOT` into the packaging service

## Task 1: Convert the Store Contract to Managed Paths

**Files:**
- Modify: `foundation/persist/workspace_store.go`
- Modify: `foundation/persist/workspace_store_test.go`
- Test: `foundation/persist/workspace_store_test.go`

- [ ] **Step 1: Write the failing schema test**

Update `foundation/persist/workspace_store_test.go` to assert the migration now creates path columns instead of blob-only storage:

```go
func TestWorkspaceMigrationStatementsUseManagedPathColumns(t *testing.T) {
	statements := workspaceMigrationStatements("sqlite")
	createTable := statements[0]

	if !strings.Contains(createTable, "workspace_path TEXT NOT NULL") {
		t.Fatalf("expected workspace_path column, got %q", createTable)
	}
	if !strings.Contains(createTable, "bundle_path TEXT") {
		t.Fatalf("expected bundle_path column, got %q", createTable)
	}
}
```

- [ ] **Step 2: Run the schema test to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./foundation/persist -run TestWorkspaceMigrationStatementsUseManagedPathColumns
```

Expected: FAIL because `workspaceMigrationStatements(...)` still defines `workspace_snapshot` / `bundle_content`.

- [ ] **Step 3: Add a failing round-trip test for path metadata**

Extend `foundation/persist/workspace_store_test.go` with:

```go
func TestSQLWorkspaceStorePersistsManagedPaths(t *testing.T) {
	ctx := context.Background()
	dbURL := "file:" + filepath.Join(t.TempDir(), "workspace-store.db")

	store, err := NewSQLWorkspaceStore(ctx, dbURL)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	createdAt := time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)
	if err := store.SaveWorkspace(ctx, "ws_test", "/managed/root/ws_test/workspace", createdAt); err != nil {
		t.Fatalf("save workspace: %v", err)
	}
	if err := store.SaveBundle(ctx, "ws_test", "/managed/root/ws_test/bundle/test.tgz", createdAt.Add(time.Minute)); err != nil {
		t.Fatalf("save bundle: %v", err)
	}

	record, err := store.LoadWorkspace(ctx, "ws_test")
	if err != nil {
		t.Fatalf("load workspace: %v", err)
	}
	if record.WorkspacePath != "/managed/root/ws_test/workspace" {
		t.Fatalf("workspace path = %q", record.WorkspacePath)
	}
	if record.BundlePath != "/managed/root/ws_test/bundle/test.tgz" {
		t.Fatalf("bundle path = %q", record.BundlePath)
	}
}
```

- [ ] **Step 4: Run the round-trip test to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./foundation/persist -run TestSQLWorkspaceStorePersistsManagedPaths
```

Expected: FAIL because `SaveWorkspace` / `SaveBundle` still accept bytes and `WorkspaceRecord` still exposes `Snapshot` / `Bundle`.

- [ ] **Step 5: Implement the minimal store contract**

Update `foundation/persist/workspace_store.go` so the interface and record become:

```go
type WorkspaceStore interface {
	SaveWorkspace(ctx context.Context, workspaceID string, workspacePath string, createdAt time.Time) error
	LoadWorkspace(ctx context.Context, workspaceID string) (WorkspaceRecord, error)
	SaveBundle(ctx context.Context, workspaceID string, bundlePath string, updatedAt time.Time) error
	LoadBundle(ctx context.Context, workspaceID string) (string, error)
	DeleteWorkspace(ctx context.Context, workspaceID string) error
	DeleteExpired(ctx context.Context, cutoff time.Time) error
}

type WorkspaceRecord struct {
	WorkspaceID   string
	WorkspacePath string
	BundlePath    string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
```

and the schema/mutations become:

```go
`CREATE TABLE IF NOT EXISTS packaging_workspaces (
	workspace_id TEXT NOT NULL PRIMARY KEY,
	workspace_path TEXT NOT NULL,
	bundle_path TEXT,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
)`
```

```go
INSERT INTO packaging_workspaces (
	workspace_id,
	workspace_path,
	bundle_path,
	created_at,
	updated_at
) VALUES (?, ?, NULL, ?, ?)
ON CONFLICT(workspace_id) DO UPDATE SET
	workspace_path = excluded.workspace_path,
	bundle_path = NULL,
	updated_at = excluded.updated_at
```

```go
UPDATE packaging_workspaces
SET bundle_path = ?, updated_at = ?
WHERE workspace_id = ?
```

and:

```go
func (s *SQLWorkspaceStore) LoadBundle(ctx context.Context, workspaceID string) (string, error)
```

returns `ErrWorkspaceBundleNotFound` when `bundle_path` is empty.

- [ ] **Step 6: Run the persist package tests to verify GREEN**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./foundation/persist
```

Expected: PASS.

- [ ] **Step 7: Commit the store conversion**

```bash
cd /Users/ultraman/workspace/skill_registry
git add foundation/persist/workspace_store.go foundation/persist/workspace_store_test.go
git commit -m "Use managed workspace paths in workspace store"
```

## Task 2: Rewire Packaging Service to Reuse Managed Workspaces

**Files:**
- Modify: `business/packaging/workspace_persistence.go`
- Modify: `business/packaging/service.go`
- Modify: `business/packaging/service_test.go`
- Test: `business/packaging/service_test.go`

- [ ] **Step 1: Write the failing service test for prepare moving into the managed root**

Add to `business/packaging/service_test.go`:

```go
func TestPrepareWithWorkspaceStoreMovesWorkspaceIntoManagedRoot(t *testing.T) {
	root := t.TempDir()
	service := NewServiceWithWorkspaceStore(root, memoryWorkspaceStore{})

	resp, err := service.Prepare(PrepareRequest{
		Source: Source{Type: SourceTypeAuthoring, InlineFiles: map[string]string{
			"SKILL.md": "# Skill\n",
		}},
		Target: Target{PackageName: "@platform.skill/managed", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "managed", Description: "desc", SourceType: "native"},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	managedWorkspace := filepath.Join(root, resp.WorkspacePath, "workspace")
	if _, err := os.Stat(filepath.Join(managedWorkspace, "SKILL.md")); err != nil {
		t.Fatalf("expected managed workspace SKILL.md: %v", err)
	}
}
```

- [ ] **Step 2: Run the targeted test to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./business/packaging -run TestPrepareWithWorkspaceStoreMovesWorkspaceIntoManagedRoot
```

Expected: FAIL because `Prepare` still archives the workspace and deletes the source directory.

- [ ] **Step 3: Add the failing bundle reuse test**

Add to `business/packaging/service_test.go`:

```go
func TestBundleContentReusesPersistedBundlePath(t *testing.T) {
	root := t.TempDir()
	store := memoryWorkspaceStore{}
	service := NewServiceWithWorkspaceStore(root, store)

	prepareResp, err := service.Prepare(PrepareRequest{
		Source: Source{Type: SourceTypeAuthoring, InlineFiles: map[string]string{
			"SKILL.md": "# Skill\n",
		}},
		Target: Target{PackageName: "@platform.skill/managed", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "managed", Description: "desc", SourceType: "native"},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	_, err = service.Pack(PackRequest{
		WorkspacePath: prepareResp.WorkspacePath,
		Target:        Target{PackageName: "@platform.skill/managed", Version: "1.0.0"},
	})
	if err != nil {
		t.Fatalf("pack: %v", err)
	}

	first, err := service.BundleContent(prepareResp.WorkspacePath, Target{PackageName: "@platform.skill/managed", Version: "1.0.0"})
	if err != nil {
		t.Fatalf("first bundle read: %v", err)
	}
	second, err := service.BundleContent(prepareResp.WorkspacePath, Target{PackageName: "@platform.skill/managed", Version: "1.0.0"})
	if err != nil {
		t.Fatalf("second bundle read: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("expected stable persisted bundle content")
	}
}
```

- [ ] **Step 4: Run the bundle reuse test to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./business/packaging -run TestBundleContentReusesPersistedBundlePath
```

Expected: FAIL because `BundleContent` still expects DB blob content and may re-enter `PackContext`.

- [ ] **Step 5: Implement the managed workspace path helpers**

Replace the archive/materialize logic in `business/packaging/workspace_persistence.go` with helpers shaped like:

```go
func (s *Service) persistManagedWorkspace(ctx context.Context, workspacePath string) (string, string, error)
func (s *Service) managedWorkspacePath(workspaceID string) string
func (s *Service) managedBundlePath(workspaceID string, target Target) string
func ensureManagedPathWithinRoot(root string, path string) error
```

Key implementation rules:

```go
managedRoot := filepath.Join(s.workspaceRoot, workspaceID)
managedWorkspace := filepath.Join(managedRoot, "workspace")
if err := os.MkdirAll(managedRoot, 0o755); err != nil { ... }
if err := os.Rename(workspacePath, managedWorkspace); err != nil { ... }
if err := s.workspaceStore.SaveWorkspace(ctx, workspaceID, managedWorkspace, time.Now().UTC()); err != nil { ... }
```

- [ ] **Step 6: Implement the service flow changes**

Update `business/packaging/service.go` so:

```go
type Service struct {
	workspaceRoot   string
	workspaceSource workspaceSourceModule
	bundleRules     bundleRuleModule
	workspaceStore  persist.WorkspaceStore
}
```

`NewServiceWithWorkspaceStore(...)` becomes:

```go
func NewServiceWithWorkspaceStore(tempRoot string, store persist.WorkspaceStore) *Service {
	if strings.TrimSpace(tempRoot) == "" {
		tempRoot = os.TempDir()
	}
	return &Service{
		workspaceRoot:   tempRoot,
		workspaceSource: newWorkspaceSourceModule(tempRoot, 24*time.Hour),
		bundleRules:     newBundleRuleModule(),
		workspaceStore:  store,
	}
}
```

and the main call sites change to:

```go
if s.workspaceStore != nil {
	handle, _, persistErr := s.persistManagedWorkspace(ctx, workspacePath)
	...
	workspacePath = handle
}
```

```go
record, err := s.workspaceStore.LoadWorkspace(ctx, workspaceID)
...
return record.WorkspacePath, func() {}, nil
```

```go
bundlePath := s.managedBundlePath(input.WorkspacePath, input.Target)
...
if saveErr := s.workspaceStore.SaveBundle(ctx, input.WorkspacePath, bundlePath, time.Now().UTC()); saveErr != nil { ... }
```

```go
bundlePath, err := s.workspaceStore.LoadBundle(ctx, workspacePath)
if err == nil {
	return os.ReadFile(bundlePath)
}
```

```go
record, err := s.workspaceStore.LoadWorkspace(ctx, workspacePath)
...
if err := os.RemoveAll(filepath.Dir(record.WorkspacePath)); err != nil { ... }
```

- [ ] **Step 7: Run the targeted business tests to verify GREEN**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./business/packaging -run 'TestPrepareWithWorkspaceStoreMovesWorkspaceIntoManagedRoot|TestBundleContentReusesPersistedBundlePath'
```

Expected: PASS.

- [ ] **Step 8: Run the full packaging package tests**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./business/packaging
```

Expected: PASS.

- [ ] **Step 9: Commit the service rewiring**

```bash
cd /Users/ultraman/workspace/skill_registry
git add business/packaging/workspace_persistence.go business/packaging/service.go business/packaging/service_test.go
git commit -m "Reuse managed packaging workspaces"
```

## Task 3: Wire the Configured Shared Workspace Root

**Files:**
- Modify: `cmd/skill-registry/main.go`
- Test: `go test ./cmd/skill-registry ./business/packaging ./foundation/persist`

- [ ] **Step 1: Write the failing configuration expectation in the integration seam test**

If there is no suitable main test, add a focused constructor test in `business/packaging/service_test.go`:

```go
func TestNewServiceWithWorkspaceStoreUsesConfiguredRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "managed-root")
	service := NewServiceWithWorkspaceStore(root, memoryWorkspaceStore{})

	if service.workspaceRoot != root {
		t.Fatalf("workspace root = %q", service.workspaceRoot)
	}
}
```

- [ ] **Step 2: Run the focused test to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./business/packaging -run TestNewServiceWithWorkspaceStoreUsesConfiguredRoot
```

Expected: FAIL if `Service` still has no dedicated `workspaceRoot` field or if the constructor does not preserve it.

- [ ] **Step 3: Wire the env var in `main.go`**

Update `cmd/skill-registry/main.go`:

```go
workspaceRoot := envOrDefault("SKILL_REGISTRY_WORKSPACE_ROOT", filepath.Join("data", "packaging-workspaces"))
```

and:

```go
Packaging: packaging.NewServiceWithWorkspaceStore(workspaceRoot, storeSet.Workspace),
```

- [ ] **Step 4: Run the constructor test to verify GREEN**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./business/packaging -run TestNewServiceWithWorkspaceStoreUsesConfiguredRoot
```

Expected: PASS.

- [ ] **Step 5: Run the final regression suite**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./cmd/skill-registry ./business/packaging ./foundation/persist
```

Expected: PASS.

- [ ] **Step 6: Commit the config wiring**

```bash
cd /Users/ultraman/workspace/skill_registry
git add cmd/skill-registry/main.go business/packaging/service_test.go
git commit -m "Configure shared packaging workspace root"
```
