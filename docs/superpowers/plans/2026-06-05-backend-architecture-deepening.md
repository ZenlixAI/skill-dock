# Backend Architecture Deepening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deepen the backend modules so the HTTP Adapter, Catalog persistence facade, and Packaging facade expose smaller interfaces with better locality and better test surfaces.

**Architecture:** Keep the external HTTP and Go package interfaces stable while splitting behavior behind them into deeper modules. Refactor in three passes: `catalogapi` workflow modules first, Catalog read/write seams second, Packaging internals third, with behavior locked by tests before each change.

**Tech Stack:** Go, `net/http`, `testing`, existing in-memory and SQL adapters

**Status:** Implemented on 2026-06-05. Result summary: `docs/superpowers/specs/2026-06-05-backend-architecture-deepening-summary.zh-CN.md`

**Execution Notes:**
- Focused implementation and package-level GREEN verification are complete.
- Some RED checkpoints from the original TDD plan are not preserved as explicit command history in this document.
- Full-suite `go test ./...` verification completed successfully on 2026-06-05.

---

## File Map

- Modify: `app/catalogapi/server.go`
  - Shrink `Server` into a route/middleware Adapter
- Create: `app/catalogapi/catalog_read_module.go`
  - Hold Catalog read use cases behind a small interface
- Create: `app/catalogapi/governance_module.go`
  - Hold governance mutation use cases behind a small interface
- Create: `app/catalogapi/packaging_module.go`
  - Hold packaging use cases, idempotency, and packaging audit behavior
- Create: `app/catalogapi/session_module.go`
  - Hold session login/logout/current-principal behavior
- Modify: `app/catalogapi/server_test.go`
  - Move most behavior assertions to module-level tests and keep only adapter coverage here
- Create: `app/catalogapi/packaging_module_test.go`
- Create: `app/catalogapi/session_module_test.go`
- Create: `app/catalogapi/catalog_read_module_test.go`
- Create: `app/catalogapi/governance_module_test.go`
- Modify: `business/catalog/repository.go`
  - Keep compatibility facade while exposing narrower internal seams
- Modify: `business/catalog/catalog_repository.go`
  - Delegate reads and aggregate mutations to deeper modules
- Create: `business/catalog/read_module.go`
- Create: `business/catalog/write_module.go`
- Create: `business/catalog/query_store.go`
- Modify: `business/catalog/aggregate_state_store.go`
- Modify: `business/catalog/sql_state_store.go`
- Modify: `business/catalog/memory_aggregate_state_store.go`
- Modify: `business/catalog/catalog_repository_test.go`
- Modify: `business/catalog/sql_repository_test.go`
- Create: `business/catalog/read_module_test.go`
- Create: `business/catalog/write_module_test.go`
- Modify: `business/packaging/service.go`
  - Keep compatibility facade while delegating to source/workspace and bundle-rule modules
- Create: `business/packaging/workspace_source_module.go`
- Create: `business/packaging/bundle_rule_module.go`
- Modify: `business/packaging/service_test.go`
- Create: `business/packaging/workspace_source_module_test.go`
- Create: `business/packaging/bundle_rule_module_test.go`

### Task 1: Lock `catalogapi` Behavior With Focused Failing Tests

**Files:**
- Modify: `app/catalogapi/server_test.go`
- Create: `app/catalogapi/packaging_module_test.go`
- Create: `app/catalogapi/session_module_test.go`
- Create: `app/catalogapi/catalog_read_module_test.go`
- Create: `app/catalogapi/governance_module_test.go`

- [x] **Step 1: Add a failing Adapter-level test that asserts route wiring still works**

```go
func TestServerRoutesSkillsThroughCatalogReadModule(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
}
```

- [x] **Step 2: Add a failing packaging module test for idempotent replay**

```go
func TestPackagingModuleReplaysCachedPrepareResponse(t *testing.T) {
	module := newPackagingModuleForTest(t)
	payload := packaging.PrepareRequest{
		Target: packaging.Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
	}

	first, firstStatus, firstErr := module.prepare(context.Background(), auth.Principal{Role: "authoring"}, payload)
	second, secondStatus, secondErr := module.prepare(context.Background(), auth.Principal{Role: "authoring"}, payload)

	if firstErr != nil || secondErr != nil {
		t.Fatalf("expected cached success, got firstErr=%v secondErr=%v", firstErr, secondErr)
	}
	if firstStatus != http.StatusOK || secondStatus != http.StatusOK {
		t.Fatalf("expected status 200/200, got %d/%d", firstStatus, secondStatus)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expected cached replay body to match")
	}
}
```

- [x] **Step 3: Add a failing session module test for current principal projection**

```go
func TestSessionModuleReturnsCurrentPrincipalView(t *testing.T) {
	module := newSessionModuleForTest(t)
	principal := auth.Principal{Role: "publisher", Actor: "publisher", AuthMechanism: "legacy_token"}

	response := module.current(principal)

	if response.Role != "publisher" {
		t.Fatalf("expected role publisher, got %q", response.Role)
	}
	if response.AuthMechanism != "legacy_token" {
		t.Fatalf("expected auth mechanism legacy_token, got %q", response.AuthMechanism)
	}
}
```

- [ ] **Step 4: Run the focused `catalogapi` tests to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./app/catalogapi
```

Expected: FAIL because the new module tests reference behavior and seams that do not exist yet.

### Task 2: Deepen `catalogapi.Server` Into Workflow Modules

**Files:**
- Modify: `app/catalogapi/server.go`
- Create: `app/catalogapi/catalog_read_module.go`
- Create: `app/catalogapi/governance_module.go`
- Create: `app/catalogapi/packaging_module.go`
- Create: `app/catalogapi/session_module.go`

- [x] **Step 1: Introduce workflow module structs and make `Server` hold them**

Add shapes like:

```go
type catalogReadModule struct {
	repo       catalog.Repository
	authorizer *auth.Authorizer
}

type governanceModule struct {
	repo       catalog.Repository
	authorizer *auth.Authorizer
	audit      *AuditRecorder
}

type packagingModule struct {
	service      *packaging.Service
	idempotency  persist.IdempotencyStore
	audit        *AuditRecorder
	cacheTTL     time.Duration
	cacheEntries map[string]cachedPackagingResponse
}

type sessionModule struct {
	authorizer     *auth.Authorizer
	sessionManager *auth.SessionManager
}
```

- [x] **Step 2: Move session use cases behind `sessionModule`**

Implement methods like:

```go
func (m *sessionModule) current(principal auth.Principal) sessionResponse { ... }
func (m *sessionModule) login(request *http.Request, username string, password string) (auth.Principal, *http.Cookie, bool) { ... }
func (m *sessionModule) logoutCookie(request *http.Request) *http.Cookie { ... }
```

- [x] **Step 3: Move packaging idempotency and audit orchestration behind `packagingModule`**

Implement methods like:

```go
func (m *packagingModule) prepare(ctx context.Context, principal auth.Principal, payload packaging.PrepareRequest) (any, int, *catalog.Error) { ... }
func (m *packagingModule) validate(ctx context.Context, principal auth.Principal, payload packaging.ValidateRequest) (any, int, *catalog.Error) { ... }
func (m *packagingModule) pack(ctx context.Context, principal auth.Principal, payload packaging.PackRequest) (any, int, *catalog.Error) { ... }
```

- [x] **Step 4: Make `Server` route handlers delegate to workflow modules**

Replace in-place behavior with thin handlers:

```go
func (s *Server) handlePackagingPrepare(writer http.ResponseWriter, request *http.Request) {
	handlePackagingEndpoint(s, writer, request, s.packagingModule.prepare)
}
```

and keep `Server` responsible only for route wiring, shared middleware, and request metadata.

- [x] **Step 5: Run the `catalogapi` tests to verify GREEN**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./app/catalogapi
```

Expected: PASS with the new module tests and the existing HTTP behavior preserved.

### Task 3: Split Catalog Reads From `Agent Skill Aggregate` Mutations

**Files:**
- Modify: `business/catalog/repository.go`
- Modify: `business/catalog/catalog_repository.go`
- Create: `business/catalog/read_module.go`
- Create: `business/catalog/write_module.go`
- Create: `business/catalog/query_store.go`
- Modify: `business/catalog/aggregate_state_store.go`
- Modify: `business/catalog/sql_state_store.go`
- Modify: `business/catalog/memory_aggregate_state_store.go`
- Modify: `business/catalog/catalog_repository_test.go`
- Create: `business/catalog/read_module_test.go`
- Create: `business/catalog/write_module_test.go`

- [x] **Step 1: Add failing tests for a read seam and a write seam**

```go
func TestReadModuleResolvesInstallTargetFromQueryableState(t *testing.T) {
	store := NewMemoryAggregateStateStore(SampleData(), MemoryAggregateStateStoreOptions{})
	module := newReadModule(store)

	resolution, err := module.ResolveInstallTarget(context.Background(), "@platform.skill/pdf-skill", "")
	if err != nil {
		t.Fatalf("resolve install target: %v", err)
	}
	if resolution.PackageName != "@platform.skill/pdf-skill" {
		t.Fatalf("unexpected package %q", resolution.PackageName)
	}
}

func TestWriteModuleAppliesVersionGovernanceChangeThroughAggregateState(t *testing.T) {
	store := NewMemoryAggregateStateStore(SampleData(), MemoryAggregateStateStoreOptions{})
	module := newWriteModule(store, NewStaticArtifactSource(SampleData().Artifacts))

	response, err := module.ApplyVersionGovernanceChange(context.Background(), "@platform.skill/pdf-skill", "1.2.0", VersionGovernanceChangeInput{
		Action: GovernanceActionYank,
		Reason: "security issue",
	}, "user_123")
	if err != nil {
		t.Fatalf("apply governance change: %v", err)
	}
	if response.ResultingState.VersionStatus == nil || *response.ResultingState.VersionStatus != VersionStatusYanked {
		t.Fatalf("unexpected resulting state: %+v", response.ResultingState)
	}
}
```

- [ ] **Step 2: Run the catalog tests to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./business/catalog
```

Expected: FAIL because the read/write modules and query seam are not defined yet.

- [x] **Step 3: Introduce a query seam and implement the read module**

Add:

```go
type QueryStore interface {
	ListSkills(ctx context.Context, query SkillQuery) (SkillsPage, error)
	ListCuratedSkills(ctx context.Context, pageSize int, cursor int) (CuratedSkillsPage, error)
	GetSkill(ctx context.Context, packageName string) (SkillDetail, error)
	ListVersions(ctx context.Context, query VersionQuery) (VersionsPage, error)
	GetVersion(ctx context.Context, packageName string, version string) (SkillVersionDetail, error)
	ListGovernanceChanges(ctx context.Context, query GovernanceQuery) (GovernancePage, error)
	ResolveInstallTarget(ctx context.Context, packageName string, version string) (InstallResolution, error)
}
```

and wire `readModule` against it.

- [x] **Step 4: Restrict aggregate mutation behavior to the write module**

Move `CreateReleaseRecord`, `ApplySkillGovernanceChange`, `ApplyVersionGovernanceChange`, and aggregate retry logic into `writeModule`.

- [x] **Step 5: Keep `CatalogRepository` as a compatibility facade and verify GREEN**

Implement delegation:

```go
type CatalogRepository struct {
	reads  *ReadModule
	writes *WriteModule
}
```

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./business/catalog
```

Expected: PASS with both old facade tests and the new seam tests green.

### Task 4: Split Packaging Workspace/Source From Bundle Rules

**Files:**
- Modify: `business/packaging/service.go`
- Create: `business/packaging/workspace_source_module.go`
- Create: `business/packaging/bundle_rule_module.go`
- Modify: `business/packaging/service_test.go`
- Create: `business/packaging/workspace_source_module_test.go`
- Create: `business/packaging/bundle_rule_module_test.go`

- [x] **Step 1: Add failing tests for workspace/source and bundle-rule seams**

```go
func TestWorkspaceSourceModuleMaterializesPlatformAuthoringFiles(t *testing.T) {
	module := newWorkspaceSourceModule(t.TempDir(), 24*time.Hour)

	workspacePath, warnings, err := module.prepareWorkspace(PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"commands/run.txt": "echo hi\n",
			},
		},
		Target: Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Zoom Out", Description: "Broader context"},
	})
	if err != nil {
		t.Fatalf("prepare workspace: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if _, err := os.Stat(filepath.Join(workspacePath, "commands", "run.txt")); err != nil {
		t.Fatalf("expected materialized file: %v", err)
	}
}

func TestBundleRuleModuleValidatesAndPacksManagedWorkspace(t *testing.T) {
	service := NewService(t.TempDir())
	prepareResp, err := service.Prepare(PrepareRequest{
		Source: Source{Type: "platform_authoring", Files: map[string]string{"commands/run.txt": "echo hi\n"}},
		Target: Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "Zoom Out", Description: "Broader context"},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	module := newBundleRuleModule()
	validateResp, err := module.validateWorkspace(prepareResp.WorkspacePath, Target{PackageName: "@platform.skill/zoom-out", Version: "1.0.0"})
	if err != nil {
		t.Fatalf("validate workspace: %v", err)
	}
	if len(validateResp.Errors) != 0 {
		t.Fatalf("expected valid workspace, got %v", validateResp.Errors)
	}
}
```

- [ ] **Step 2: Run the packaging tests to verify RED**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./business/packaging
```

Expected: FAIL because the internal modules do not exist yet.

- [x] **Step 3: Extract workspace/source responsibilities**

Move source acquisition and workspace lifecycle behind methods like:

```go
type workspaceSourceModule struct {
	tempRoot     string
	workspaceTTL time.Duration
}

func (m *workspaceSourceModule) prepareWorkspace(input PrepareRequest) (string, []string, error) { ... }
func (m *workspaceSourceModule) cleanupWorkspace(workspacePath string) error { ... }
func (m *workspaceSourceModule) bundlePath(workspacePath string, target Target) (string, error) { ... }
```

- [x] **Step 4: Extract bundle-rule responsibilities**

Move validation, import-report, manifest digest, component scanning, and bundle archive rules behind methods like:

```go
type bundleRuleModule struct{}

func (m *bundleRuleModule) finalizePreparedWorkspace(workspacePath string, input PrepareRequest, warnings []string) (PrepareResponse, error) { ... }
func (m *bundleRuleModule) validateWorkspace(workspacePath string, target Target) (validationResult, error) { ... }
func (m *bundleRuleModule) packWorkspace(workspacePath string, target Target, includeBundle bool) (PackResponse, error) { ... }
```

- [x] **Step 5: Keep `Service` as the compatibility facade and verify GREEN**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./business/packaging
```

Expected: PASS with old service tests plus the new internal module tests green.

### Task 5: Run Cross-Module Verification

**Files:**
- No code changes

- [x] **Step 1: Run the focused backend test suites**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./app/catalogapi ./business/catalog ./business/packaging
```

Expected: PASS with all targeted backend refactor tests green.

- [ ] **Step 2: Run the full Go test suite**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && go test ./...
```

Expected: PASS, or a clear list of unrelated failures if the workspace already had them.

- [ ] **Step 3: Review the diff for seam depth and API compatibility**

Run:

```bash
cd /Users/ultraman/workspace/skill_registry && git diff -- app/catalogapi business/catalog business/packaging docs/superpowers/plans/2026-06-05-backend-architecture-deepening.md
```

Expected: `Server`, `CatalogRepository`, and `packaging.Service` remain compatibility facades while behavior lives in deeper modules.
