# Unified Backend Architecture Deepening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deepen the backend across Catalog reads, Packaging runtime, Published Release orchestration, the HTTP adapter, and Catalog persistence while keeping HTTP, CLI, and public Go facade contracts backward-compatible.

**Architecture:** Keep `catalog.Repository`, `packaging.Service`, and `catalogapi.NewServer(ServerConfig)` as compatibility facades. Push real behavior into deeper internal seams in five passes: SQL-backed query shaping first, workspace adapter extraction second, publish workflow state modeling third, HTTP adapter slimming fourth, and aggregate persistence deepening last.

**Tech Stack:** Go, `net/http`, SQL-backed stores, package-local tests in `business/catalog`, `business/packaging`, and `app/catalogapi`

---

## File Map

### Catalog

- `business/catalog/sql_state_store.go`
  - Move list/filter/sort/page semantics into SQL-backed query helpers and reduce in-memory shaping.
- `business/catalog/read_module.go`
  - Keep install-resolution policy and read-side rule shaping only.
- `business/catalog/query_store.go`
  - Tighten the query seam to reflect query-shaped results rather than row transport.
- `business/catalog/write_module.go`
  - Preserve aggregate mutation rules while the persistence implementation changes underneath.
- `business/catalog/sql_repository_test.go`
  - Integration-style SQL repository coverage for list/read/write behavior.
- `business/catalog/read_module_test.go`
  - Focused read-policy tests.
- `business/catalog/write_module_test.go`
  - Focused write-policy and aggregate mutation tests.

### Packaging

- `business/packaging/workspace_runtime_module.go`
  - Delegate runtime orchestration to explicit workspace adapters instead of inline `workspaceStore != nil` branching.
- `business/packaging/workspace_adapter.go`
  - New internal workspace adapter seam shared by managed and ephemeral implementations.
- `business/packaging/service.go`
  - Continue as compatibility facade and wire runtime + publish workflow.
- `business/packaging/published_release_workflow.go`
  - Replace error-code-driven control flow with internal workflow state.
- `business/packaging/workspace_runtime_module_test.go`
  - Runtime behavior and adapter boundary coverage.
- `business/packaging/published_release_workflow_test.go`
  - Publish recovery, replay, and conflict-path tests.
- `business/packaging/service_test.go`
  - Compatibility-facade smoke coverage.

### Catalog API

- `app/catalogapi/server.go`
  - Shrink toward route wiring, shared middleware, request metadata, and shared observability only.
- `app/catalogapi/catalog_read_module.go`
  - Extend/adjust read-side HTTP workflow delegation.
- `app/catalogapi/packaging_access_module.go`
  - Continue as packaging access workflow seam.
- `app/catalogapi/read_api_module.go`
  - New module for read subtree query parsing and delegation.
- `app/catalogapi/system_module.go`
  - New module for readiness/health/system workflows.
- `app/catalogapi/mock_registry_module.go`
  - New module for mock npm HTTP adapter behavior.
- `app/catalogapi/server_test.go`
  - Route/middleware smoke coverage only.
- `app/catalogapi/server_workflow_seam_test.go`
  - Focused coverage for new workflow seams.

## Task 1: Deepen the Catalog Query Store

**Files:**
- Modify: `business/catalog/sql_state_store.go`
- Modify: `business/catalog/read_module.go`
- Modify: `business/catalog/query_store.go`
- Modify: `business/catalog/sql_repository_test.go`
- Modify: `business/catalog/read_module_test.go`

- [ ] **Step 1: Write the failing SQL list-shaping tests**

Add these tests to `business/catalog/sql_repository_test.go`:

```go
func TestSQLRepositoryListSkillsAppliesSortAndPageInStore(t *testing.T) {
	repo := newSQLRepositoryForTest(t, SampleData())

	page, err := repo.ListSkills(context.Background(), SkillQuery{
		Sort:     "package_name_asc",
		PageSize: 1,
		Cursor:   1,
	})
	if err != nil {
		t.Fatalf("list skills: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected exactly one item, got %d", len(page.Items))
	}
	if page.Items[0].PackageName != "@platform.skill/zoom-out-skill" {
		t.Fatalf("expected second lexicographic package, got %q", page.Items[0].PackageName)
	}
}

func TestSQLRepositoryListGovernanceChangesPagesInStore(t *testing.T) {
	repo := newSQLRepositoryForTest(t, SampleData())

	_, err := repo.ApplyVersionGovernanceChange(context.Background(), "@platform.skill/pdf-skill", "1.1.0", VersionGovernanceChangeInput{
		Action: GovernanceActionDeprecate,
		Reason: "legacy",
	}, "operator")
	if err != nil {
		t.Fatalf("seed governance change: %v", err)
	}
	_, err = repo.ApplyVersionGovernanceChange(context.Background(), "@platform.skill/pdf-skill", "1.2.0", VersionGovernanceChangeInput{
		Action: GovernanceActionYank,
		Reason: "security",
	}, "operator")
	if err != nil {
		t.Fatalf("seed governance change: %v", err)
	}

	page, err := repo.ListGovernanceChanges(context.Background(), GovernanceQuery{
		PackageName: "@platform.skill/pdf-skill",
		PageSize:    1,
		Cursor:      1,
	})
	if err != nil {
		t.Fatalf("list governance changes: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected one governance item, got %d", len(page.Items))
	}
}
```

- [ ] **Step 2: Run the focused catalog tests to verify RED**

Run:

```bash
env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/catalog -run 'TestSQLRepositoryListSkillsAppliesSortAndPageInStore|TestSQLRepositoryListGovernanceChangesPagesInStore'
```

Expected: FAIL because current implementation still performs list shaping in memory after broad queries.

- [ ] **Step 3: Implement query-shaped SQL helpers**

In `business/catalog/sql_state_store.go`, replace the current broad-load-then-sort flow with SQL-limited helpers similar to:

```go
func (s *SQLStateStore) ListSkills(ctx context.Context, query SkillQuery) (SkillsPage, error) {
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		return SkillsPage{}, NewError(400, "invalid_request", "pageSize must be less than or equal to 100.", map[string]any{"pageSize": query.PageSize})
	}

	sortKey := normalizedSkillsSort(query.Sort, query.Query)
	rows, err := s.db.QueryContext(ctx, bindSQL(s.dialect, s.listSkillsSQL(query, sortKey)), s.listSkillsArgs(query, sortKey)...)
	if err != nil {
		return SkillsPage{}, err
	}
	defer rows.Close()

	items := make([]SkillSummary, 0, query.PageSize+1)
	for rows.Next() {
		item, err := scanSkillSummary(rows)
		if err != nil {
			return SkillsPage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return SkillsPage{}, err
	}
	return pageSkillSummaries(items, query.Cursor, query.PageSize), nil
}

func pageSkillSummaries(items []SkillSummary, cursor int, pageSize int) SkillsPage {
	hasNextPage := len(items) > pageSize
	if hasNextPage {
		items = items[:pageSize]
	}
	var nextCursor *string
	if hasNextPage {
		value := strconv.Itoa(cursor + len(items))
		nextCursor = &value
	}
	return SkillsPage{
		Items: items,
		PageInfo: PageInfo{
			NextCursor: nextCursor,
		},
	}
}
```

And define SQL ordering explicitly:

```go
func normalizedSkillsSort(sortValue string, query string) string {
	if sortValue == "" {
		if strings.TrimSpace(query) != "" {
			return "relevance"
		}
		return "updated_at_desc"
	}
	return sortValue
}
```

Also update `ListVersions` and `ListGovernanceChanges` so the SQL query itself applies `ORDER BY`, `LIMIT`, and `OFFSET`.

- [ ] **Step 4: Keep `ReadModule` as a policy seam**

Leave `business/catalog/read_module.go` in the following shape:

```go
func (m *ReadModule) ResolveInstallTarget(ctx context.Context, packageName string, version string) (InstallResolution, error) {
	skill, err := m.store.GetSkill(ctx, packageName)
	if err != nil {
		return InstallResolution{}, err
	}
	if skill.Status == SkillStatusRetired {
		return InstallResolution{}, NewError(409, "skill_retired", "Skill is retired.", map[string]any{"packageName": packageName})
	}

	versions, err := m.store.ListPackageVersions(ctx, packageName)
	if err != nil {
		return InstallResolution{}, err
	}
	target, err := m.resolveInstallVersion(packageName, version, versions)
	if err != nil {
		return InstallResolution{}, err
	}
	return InstallResolution{
		PackageName:         target.PackageName,
		Version:             target.Version,
		NPMRegistry:         target.Registry.NPMRegistry,
		Integrity:           target.Registry.Integrity,
		TarballURL:          target.Registry.TarballURL,
		SkillManifestDigest: target.Bundle.SkillManifestDigest,
		VersionStatus:       target.VersionStatus,
	}, nil
}
```

- [ ] **Step 5: Run package tests to verify GREEN**

Run:

```bash
env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/catalog
```

Expected: PASS

- [ ] **Step 6: Commit the Catalog query deepening**

```bash
git add business/catalog/sql_state_store.go business/catalog/read_module.go business/catalog/query_store.go business/catalog/sql_repository_test.go business/catalog/read_module_test.go
git commit -m "refactor: deepen catalog query store"
```

## Task 2: Extract explicit workspace adapters behind Packaging Workspace Runtime

**Files:**
- Create: `business/packaging/workspace_adapter.go`
- Modify: `business/packaging/workspace_runtime_module.go`
- Modify: `business/packaging/service.go`
- Modify: `business/packaging/workspace_runtime_module_test.go`
- Modify: `business/packaging/service_test.go`

- [ ] **Step 1: Write the failing runtime adapter tests**

Add to `business/packaging/workspace_runtime_module_test.go`:

```go
func TestPackagingWorkspaceRuntimeUsesManagedAdapterWhenWorkspaceStoreExists(t *testing.T) {
	store := newMemoryWorkspaceStore()
	runtime := newPackagingWorkspaceRuntime(t.TempDir(), store)

	response, err := runtime.Prepare(context.Background(), PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"SKILL.md": "skill body",
				"docs/README.md": "# docs",
			},
		},
		Target:   Target{PackageName: "@platform.skill/runtime-adapter", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "runtime-adapter", Description: "desc", SourceType: catalog.SourceTypeNative},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if !strings.HasPrefix(response.WorkspacePath, "workspace_") {
		t.Fatalf("expected managed workspace handle, got %q", response.WorkspacePath)
	}
}

func TestPackagingWorkspaceRuntimeUsesEphemeralAdapterWithoutWorkspaceStore(t *testing.T) {
	runtime := newPackagingWorkspaceRuntime(t.TempDir(), nil)

	response, err := runtime.Prepare(context.Background(), PrepareRequest{
		Source: Source{
			Type: "platform_authoring",
			Files: map[string]string{
				"SKILL.md": "skill body",
				"docs/README.md": "# docs",
			},
		},
		Target:   Target{PackageName: "@platform.skill/runtime-ephemeral", Version: "1.0.0"},
		Metadata: Metadata{DisplayName: "runtime-ephemeral", Description: "desc", SourceType: catalog.SourceTypeNative},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if !strings.Contains(response.WorkspacePath, "skill-packaging-") {
		t.Fatalf("expected filesystem workspace path, got %q", response.WorkspacePath)
	}
}
```

- [ ] **Step 2: Run the focused packaging tests to verify RED**

Run:

```bash
env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/packaging -run 'TestPackagingWorkspaceRuntimeUsesManagedAdapterWhenWorkspaceStoreExists|TestPackagingWorkspaceRuntimeUsesEphemeralAdapterWithoutWorkspaceStore'
```

Expected: FAIL because the runtime still branches inline instead of using explicit adapters.

- [ ] **Step 3: Add the internal workspace adapter seam**

Create `business/packaging/workspace_adapter.go`:

```go
package packaging

import (
	"context"
	"time"
)

type workspaceAdapter interface {
	prepareHandle(ctx context.Context, workspacePath string) (string, error)
	validateWorkspace(ctx context.Context, workspacePath string) (workspaceMaterialization, error)
	loadBundlePath(ctx context.Context, workspaceHandle string, target Target) (string, error)
	saveValidationCache(ctx context.Context, workspaceHandle string, cache validationResult) error
	loadValidationCache(ctx context.Context, workspaceHandle string, target Target) (validationResult, bool, error)
	loadPackageMetadata(ctx context.Context, workspaceHandle string) (map[string]any, bool, error)
	savePackageMetadata(ctx context.Context, workspaceHandle string, payload []byte) error
	cleanupWorkspace(ctx context.Context, workspaceHandle string) error
	pruneExpired(ctx context.Context, ttl time.Duration) error
}

type workspaceMaterialization struct {
	handle           string
	workspacePath    string
	cleanup          func()
	packageJSONCache []byte
}
```

- [ ] **Step 4: Refactor runtime to choose one adapter**

Refactor `business/packaging/workspace_runtime_module.go`:

```go
type packagingWorkspaceRuntime struct {
	workspace   *packagingWorkspaceBacking
	bundleRules bundleRuleModule
	adapter     workspaceAdapter
}

func newPackagingWorkspaceRuntimeWithBacking(workspace *packagingWorkspaceBacking) *packagingWorkspaceRuntime {
	runtime := &packagingWorkspaceRuntime{
		workspace:   workspace,
		bundleRules: newBundleRuleModule(),
	}
	if workspace.workspaceStore != nil {
		runtime.adapter = newManagedWorkspaceAdapter(workspace)
	} else {
		runtime.adapter = newEphemeralWorkspaceAdapter(workspace)
	}
	return runtime
}
```

Then replace `if r.workspace.workspaceStore != nil` branches with adapter delegation:

```go
func (r *packagingWorkspaceRuntime) Prepare(ctx context.Context, input PrepareRequest) (PrepareResponse, error) {
	workspacePath, err := r.workspace.workspaceSource.prepare(input)
	if err != nil {
		return PrepareResponse{}, err
	}
	outcome, err := r.bundleRules.prepareWorkspace(workspacePath, input)
	if err != nil {
		return PrepareResponse{}, err
	}
	handle, err := r.adapter.prepareHandle(ctx, workspacePath)
	if err != nil {
		return PrepareResponse{}, err
	}
	return PrepareResponse{
		WorkspacePath:  handle,
		PackageName:    input.Target.PackageName,
		Version:        input.Target.Version,
		Warnings:       outcome.Warnings,
		WarningDetails: warningDetails(outcome.Warnings),
		ImportReport:   outcome.ImportReport,
		Errors:         []string{},
	}, nil
}
```

- [ ] **Step 5: Verify runtime and service compatibility**

Run:

```bash
env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/packaging
```

Expected: PASS

- [ ] **Step 6: Commit the runtime adapter deepening**

```bash
git add business/packaging/workspace_adapter.go business/packaging/workspace_runtime_module.go business/packaging/service.go business/packaging/workspace_runtime_module_test.go business/packaging/service_test.go
git commit -m "refactor: extract packaging workspace adapters"
```

## Task 3: Replace Published Release error-code control flow with internal workflow state

**Files:**
- Modify: `business/packaging/published_release_workflow.go`
- Modify: `business/packaging/published_release_workflow_test.go`
- Modify: `business/packaging/service.go`

- [ ] **Step 1: Write failing publish workflow tests**

Add to `business/packaging/published_release_workflow_test.go`:

```go
func TestPublishedReleaseWorkflowPublishesWhenArtifactMissing(t *testing.T) {
	workflow := newPublishedReleaseWorkflow(newStubRuntime(t), stubReleaseRecordCreator{
		createReleaseRecord: func(ctx context.Context, packageName string, version string, input catalog.ReleaseRecordInput) (catalog.ReleaseRecordResult, error) {
			return catalog.ReleaseRecordResult{}, errReleaseArtifactMissing
		},
	}, &stubPackagePublisher{})

	response, err := workflow.Publish(context.Background(), PublishRequest{
		WorkspacePath: "workspace_123",
		Target:        Target{PackageName: "@platform.skill/pdf-skill", Version: "1.2.0"},
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if !response.PublishedToRegistry {
		t.Fatalf("expected publish path to touch registry")
	}
}

func TestPublishedReleaseWorkflowReplaysWhenArtifactAlreadyExists(t *testing.T) {
	calls := 0
	workflow := newPublishedReleaseWorkflow(newStubRuntime(t), stubReleaseRecordCreator{
		createReleaseRecord: func(ctx context.Context, packageName string, version string, input catalog.ReleaseRecordInput) (catalog.ReleaseRecordResult, error) {
			calls++
			if calls == 1 {
				return catalog.ReleaseRecordResult{}, errReleaseArtifactMissing
			}
			return catalog.ReleaseRecordResult{PackageName: packageName, Version: version, Created: false, IdempotentReplay: true}, nil
		},
	}, stubPackagePublisher{
		publishPackage: func(ctx context.Context, input artifactadapter.NPMPublishInput) error {
			return errPackageArtifactAlreadyExists
		},
	})

	response, err := workflow.Publish(context.Background(), PublishRequest{
		WorkspacePath: "workspace_123",
		Target:        Target{PackageName: "@platform.skill/pdf-skill", Version: "1.2.0"},
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if !response.IdempotentReplay {
		t.Fatalf("expected idempotent replay response")
	}
}
```

- [ ] **Step 2: Run the focused publish tests to verify RED**

Run:

```bash
env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/packaging -run 'TestPublishedReleaseWorkflowPublishesWhenArtifactMissing|TestPublishedReleaseWorkflowReplaysWhenArtifactAlreadyExists'
```

Expected: FAIL because the current workflow still branches on `catalog.Error.Code`.

- [ ] **Step 3: Introduce internal workflow-state results**

Refactor `business/packaging/published_release_workflow.go` to use internal sentinels:

```go
var (
	errReleaseArtifactMissing      = errors.New("release artifact missing")
	errPackageArtifactAlreadyExists = errors.New("package artifact already exists")
)

func classifyReleaseRecordError(err error) error {
	if catalogErr, ok := err.(*catalog.Error); ok {
		switch catalogErr.Code {
		case "registry_version_not_found":
			return errReleaseArtifactMissing
		}
	}
	return err
}

func classifyPublisherError(err error) error {
	if catalogErr, ok := err.(*catalog.Error); ok {
		switch catalogErr.Code {
		case "registry_version_conflict":
			return errPackageArtifactAlreadyExists
		}
	}
	return err
}
```

And move control flow to explicit state:

```go
releaseResult, err := w.releaseRecords.CreateReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
if err == nil {
	return publishResponseFromPack(packResponse, releaseResult, false), nil
}
if !errors.Is(classifyReleaseRecordError(err), errReleaseArtifactMissing) {
	return PublishResponse{}, packagingCatalogError(err)
}

publishErr := w.publisher.PublishPackage(ctx, artifactadapter.NPMPublishInput{...})
if publishErr != nil {
	if errors.Is(classifyPublisherError(publishErr), errPackageArtifactAlreadyExists) {
		releaseResult, err = w.releaseRecords.CreateReleaseRecord(ctx, input.Target.PackageName, input.Target.Version, releaseInput)
		if err == nil {
			return publishResponseFromPack(packResponse, releaseResult, false), nil
		}
	}
	return PublishResponse{}, packagingCatalogError(publishErr)
}
```

- [ ] **Step 4: Run packaging tests to verify GREEN**

Run:

```bash
env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/packaging
```

Expected: PASS

- [ ] **Step 5: Commit the publish workflow deepening**

```bash
git add business/packaging/published_release_workflow.go business/packaging/published_release_workflow_test.go business/packaging/service.go
git commit -m "refactor: model published release workflow states"
```

## Task 4: Shrink `catalogapi.Server` to a thinner adapter

**Files:**
- Create: `app/catalogapi/read_api_module.go`
- Create: `app/catalogapi/system_module.go`
- Create: `app/catalogapi/mock_registry_module.go`
- Modify: `app/catalogapi/server.go`
- Modify: `app/catalogapi/server_test.go`
- Modify: `app/catalogapi/server_workflow_seam_test.go`

- [ ] **Step 1: Add failing workflow-seam tests**

Add to `app/catalogapi/server_workflow_seam_test.go`:

```go
func TestReadAPIModuleListsSkillsFromParsedQuery(t *testing.T) {
	module := &readAPIModule{
		repo: catalog.NewMemoryRepository(catalog.SampleData()),
	}

	response, err := module.listSkills(context.Background(), auth.Principal{Role: "reader"}, url.Values{
		"sort":     []string{"package_name_asc"},
		"pageSize": []string{"1"},
	})
	if err != nil {
		t.Fatalf("list skills: %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("expected one item, got %d", len(response.Items))
	}
}

func TestSystemModuleReadyzCollectsDependencyChecks(t *testing.T) {
	module := &systemModule{
		repository:  stubPinger{err: nil},
		audit:       stubPinger{err: nil},
		idempotency: stubPinger{err: nil},
	}
	status, payload := module.readyz(context.Background())
	if status != http.StatusOK {
		t.Fatalf("expected ready status, got %d", status)
	}
	if payload["status"] != "ready" {
		t.Fatalf("unexpected payload %#v", payload)
	}
}
```

- [ ] **Step 2: Run focused catalogapi tests to verify RED**

Run:

```bash
env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./app/catalogapi -run 'TestReadAPIModuleListsSkillsFromParsedQuery|TestSystemModuleReadyzCollectsDependencyChecks'
```

Expected: FAIL because the new seams do not exist yet.

- [ ] **Step 3: Create dedicated read/system/mock-registry modules**

Create `app/catalogapi/read_api_module.go`:

```go
package catalogapi

type readAPIModule struct {
	repo catalog.Repository
}

func (m *readAPIModule) listSkills(ctx context.Context, principal auth.Principal, values url.Values) (catalog.SkillsPage, *catalog.Error) {
	sourceType, sourceTypeErr := parseSourceTypeFilter(values.Get("source_type"))
	if sourceTypeErr != nil {
		return catalog.SkillsPage{}, sourceTypeErr
	}
	includeRetired, includeRetiredErr := parseBooleanQueryParam(values.Get("include_retired"), "include_retired", false)
	if includeRetiredErr != nil {
		return catalog.SkillsPage{}, includeRetiredErr
	}
	pageSize, err := parsePageSize(values)
	if err != nil {
		return catalog.SkillsPage{}, err
	}
	cursor, err := parseCursor(values)
	if err != nil {
		return catalog.SkillsPage{}, err
	}
	return (&catalogReadModule{repo: m.repo}).listSkills(ctx, principal, catalog.SkillQuery{
		Query:          values.Get("q"),
		PackageName:    values.Get("package_name"),
		SourceType:     sourceType,
		IncludeRetired: includeRetired,
		Sort:           values.Get("sort"),
		PageSize:       pageSize,
		Cursor:         cursor,
	})
}
```

Create `app/catalogapi/system_module.go`:

```go
package catalogapi

type systemModule struct {
	repository  readinessPinger
	audit       readinessPinger
	idempotency readinessPinger
}

func (m *systemModule) readyz(ctx context.Context) (int, map[string]any) {
	checks := map[string]string{"repository": "ok", "audit": "ok", "idempotency": "ok"}
	status := http.StatusOK
	if m.repository != nil {
		if err := m.repository.Ping(ctx); err != nil {
			checks["repository"] = err.Error()
			status = http.StatusServiceUnavailable
		}
	}
	if m.audit != nil {
		if err := m.audit.Ping(ctx); err != nil {
			checks["audit"] = err.Error()
			status = http.StatusServiceUnavailable
		}
	}
	if m.idempotency != nil {
		if err := m.idempotency.Ping(ctx); err != nil {
			checks["idempotency"] = err.Error()
			status = http.StatusServiceUnavailable
		}
	}
	payload := map[string]any{"status": "ready", "checks": checks}
	if status != http.StatusOK {
		payload["status"] = "degraded"
	}
	return status, payload
}
```

- [ ] **Step 4: Delegate `Server` handlers to these modules**

Refactor `app/catalogapi/server.go` so handlers become thin:

```go
func (s *Server) handleSkills(writer http.ResponseWriter, request *http.Request) {
	if packageName := request.URL.Query().Get("package_name"); packageName != "" {
		if !s.authorizePackageAccess(writer, request, packageName) {
			return
		}
	}
	response, err := s.readAPI.listSkills(request.Context(), auth.PrincipalFromContext(request.Context()), request.URL.Query())
	if err != nil {
		writeCatalogError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusOK, response)
}

func (s *Server) handleReadyz(writer http.ResponseWriter, request *http.Request) {
	observability.SetObservedAction(writer, string(auth.ActionReadReadyz))
	observability.SetObservedAuthDecision(writer, observability.AuthDecisionAllowed)
	status, payload := s.systemModule.readyz(request.Context())
	writeJSON(writer, status, payload)
}
```

Also move mock npm request dispatch out of `server.go` into `mock_registry_module.go`.

- [ ] **Step 5: Run `catalogapi` tests to verify GREEN**

Run:

```bash
env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./app/catalogapi
```

Expected: PASS

- [ ] **Step 6: Commit the HTTP adapter slimming**

```bash
git add app/catalogapi/read_api_module.go app/catalogapi/system_module.go app/catalogapi/mock_registry_module.go app/catalogapi/server.go app/catalogapi/server_test.go app/catalogapi/server_workflow_seam_test.go
git commit -m "refactor: slim catalogapi server adapter"
```

## Task 5: Deepen Catalog aggregate persistence without changing aggregate semantics

**Files:**
- Modify: `business/catalog/sql_state_store.go`
- Modify: `business/catalog/write_module.go`
- Modify: `business/catalog/write_module_test.go`
- Modify: `business/catalog/sql_repository_test.go`

- [ ] **Step 1: Write failing persistence-shape tests**

Add to `business/catalog/sql_repository_test.go`:

```go
func TestSQLRepositoryApplyVersionGovernanceChangePreservesExistingGovernanceRows(t *testing.T) {
	repo := newSQLRepositoryForTest(t, SampleData())

	_, err := repo.ApplyVersionGovernanceChange(context.Background(), "@platform.skill/pdf-skill", "1.1.0", VersionGovernanceChangeInput{
		Action: GovernanceActionDeprecate,
		Reason: "legacy",
	}, "operator")
	if err != nil {
		t.Fatalf("first governance change: %v", err)
	}
	_, err = repo.ApplyVersionGovernanceChange(context.Background(), "@platform.skill/pdf-skill", "1.2.0", VersionGovernanceChangeInput{
		Action: GovernanceActionYank,
		Reason: "security",
	}, "operator")
	if err != nil {
		t.Fatalf("second governance change: %v", err)
	}

	rows := countRows(t, repo.(*CatalogRepository), "governance_records", "@platform.skill/pdf-skill")
	if rows < 2 {
		t.Fatalf("expected both governance rows to remain, got %d", rows)
	}
}
```

- [ ] **Step 2: Run focused persistence tests to verify RED**

Run:

```bash
env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/catalog -run 'TestSQLRepositoryApplyVersionGovernanceChangePreservesExistingGovernanceRows'
```

Expected: FAIL because current persistence rewrites the full aggregate row set.

- [ ] **Step 3: Split targeted persistence writes from full aggregate creation**

Refactor `business/catalog/sql_state_store.go`:

```go
func (s *SQLStateStore) PutAggregate(ctx context.Context, aggregate AgentSkillAggregate, expectedRevision uint64) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	currentRevision, exists, err := s.loadAggregateRevisionTx(ctx, tx, aggregate.Skill.Detail.PackageName)
	if err != nil {
		return err
	}
	switch {
	case exists && currentRevision != expectedRevision:
		return ErrAggregateRevisionConflict
	case !exists && expectedRevision != 0:
		return ErrAggregateRevisionConflict
	}

	if !exists {
		if err = s.insertAggregateTx(ctx, tx, cloneAggregate(aggregate), 1); err != nil {
			return err
		}
		err = tx.Commit()
		return err
	}

	if err = s.updateSkillSnapshotTx(ctx, tx, aggregate.Skill.Detail, aggregate.Skill.CurationReason, expectedRevision+1); err != nil {
		return err
	}
	if err = s.replaceVersionRowsTx(ctx, tx, aggregate.Skill.Detail.PackageName, aggregate.Skill.Versions); err != nil {
		return err
	}
	if err = s.upsertGovernanceRowsTx(ctx, tx, aggregate.Skill.Detail.PackageName, aggregate.Governance); err != nil {
		return err
	}
	err = tx.Commit()
	return err
}
```

Implement targeted helpers:

```go
func (s *SQLStateStore) updateSkillSnapshotTx(ctx context.Context, tx *sql.Tx, detail SkillDetail, curationReason string, revision uint64) error { ... }
func (s *SQLStateStore) replaceVersionRowsTx(ctx context.Context, tx *sql.Tx, packageName string, versions []SkillVersionDetail) error { ... }
func (s *SQLStateStore) upsertGovernanceRowsTx(ctx context.Context, tx *sql.Tx, packageName string, records []GovernanceRecord) error { ... }
```

In `upsertGovernanceRowsTx`, use row identity instead of delete-all:

```go
query := bindSQL(s.dialect, `
INSERT INTO governance_records (
	id, target_type, package_name, version, action, reason, replacement_recommendation, changed_by, changed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	target_type = excluded.target_type,
	version = excluded.version,
	action = excluded.action,
	reason = excluded.reason,
	replacement_recommendation = excluded.replacement_recommendation,
	changed_by = excluded.changed_by,
	changed_at = excluded.changed_at`)
```

- [ ] **Step 4: Run full catalog tests to verify GREEN**

Run:

```bash
env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/catalog
```

Expected: PASS

- [ ] **Step 5: Commit the persistence deepening**

```bash
git add business/catalog/sql_state_store.go business/catalog/write_module.go business/catalog/write_module_test.go business/catalog/sql_repository_test.go
git commit -m "refactor: deepen catalog aggregate persistence"
```

## Task 6: Run end-to-end verification across affected packages

**Files:**
- Verify only: `business/catalog/...`
- Verify only: `business/packaging/...`
- Verify only: `app/catalogapi/...`

- [ ] **Step 1: Run the focused package suite**

Run:

```bash
env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./business/catalog ./business/packaging ./app/catalogapi
```

Expected: PASS

- [ ] **Step 2: Run the full repository suite**

Run:

```bash
env GOCACHE=/private/tmp/skill_registry-go-build go test -count=1 ./...
```

Expected: PASS

- [ ] **Step 3: Inspect the final diff before handoff**

Run:

```bash
git status --short
git log --oneline -n 8
```

Expected:

- clean or intentionally staged worktree
- one commit per major workstream plus final verification state

- [ ] **Step 4: Commit any final compatibility fixups**

If verification required a final small fixup:

```bash
git add .
git commit -m "test: verify unified backend deepening"
```

If no fixup was needed, skip this step and leave the commit history as-is.
