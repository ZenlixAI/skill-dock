# One-Shot Backend Consolidation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Collapse the remaining backend compatibility graph onto retained `internal/*` seams by deleting `business/catalog`, deleting `business/packaging`, and shrinking `app/catalogapi` imports to retained owners.

**Architecture:** First promote the missing retained constructors into `internal/catalog/infra`, then migrate all repository/type imports away from `business/catalog` and delete that package. Next move packaging callers onto `internal/packaging/{domain,application,infra}` and delete `business/packaging`. Finish by moving `app/catalogapi` callers off the remaining helper subpackages that already have retained internal owners.

**Tech Stack:** Go 1.26, Gin, standard library `net/http`, existing Go test suite

---

## File Structure

- Create: `internal/catalog/application/provider.go`
- Create: `internal/catalog/infra/repository.go`
- Modify: `internal/catalog/infra/memory_repository.go`
- Modify: `cmd/skill-registry/main.go`
- Modify: `cmd/skill-registry/main_test.go`
- Modify: `internal/backend/runtime.go`
- Modify: `internal/backend/runtime_test.go`
- Modify: `app/catalogapi/interfaces.go`
- Modify: `app/catalogapi/server.go`
- Modify: `app/catalogapi/server_workflows.go`
- Modify: `app/catalogapi/catalog_read_module.go`
- Modify: `app/catalogapi/request_limiter.go`
- Modify: `app/catalogapi/common/params.go`
- Modify: `app/catalogapi/audit/query.go`
- Modify: `app/catalogapi/audit/query_test.go`
- Modify: `app/catalogapi/handlers/session/handlers.go`
- Modify: `app/catalogapi/server_test.go`
- Modify: `app/catalogapi/server_catalog_mutation_test.go`
- Modify: `app/catalogapi/server_workflow_seam_test.go`
- Modify: `app/catalogapi/server_packaging_access_test.go`
- Modify: `internal/catalog/http/binding.go`
- Modify: `internal/catalog/http/handlers.go`
- Modify: `internal/governance/http/routes.go`
- Modify: `internal/governance/http/routes_test.go`
- Modify: `internal/packaging/domain/types.go`
- Modify: `internal/packaging/domain/bundle_rules.go`
- Modify: `internal/packaging/application/service.go`
- Modify: `internal/packaging/application/publish_service.go`
- Modify: `internal/packaging/application/publish_service_test.go`
- Modify: `internal/packaging/http/workflow.go`
- Modify: `internal/packaging/http/handlers.go`
- Modify: `internal/packaging/http/binding.go`
- Modify: `internal/packaging/infra/workspace_adapter.go`
- Modify: `internal/packaging/infra/workspace_backing.go`
- Modify: `internal/packaging/infra/workspace_metadata.go`
- Modify: `internal/packaging/infra/workspace_pack_helpers.go`
- Modify: `internal/packaging/infra/workspace_source.go`
- Modify: `internal/platform/artifact/publisher.go`
- Modify: `internal/platform/artifact/sources.go`
- Modify: `internal/platform/artifact/sources_test.go`
- Modify: `internal/platform/auth/authorizer.go`
- Modify: `internal/platform/http/middleware/auth.go`
- Modify: `internal/platform/http/request_limiter.go`
- Modify: `internal/platform/http/response.go`
- Modify: `internal/platform/persistence/audit_store.go`
- Delete: `business/catalog/*.go`
- Delete: `business/packaging/*.go`

### Task 1: Promote the retained catalog seam and delete `business/catalog`

**Files:**
- Create: `internal/catalog/application/provider.go`
- Create: `internal/catalog/infra/repository.go`
- Modify: `internal/catalog/infra/memory_repository.go`
- Modify: `cmd/skill-registry/main.go`
- Modify: `cmd/skill-registry/main_test.go`
- Modify: `internal/backend/runtime.go`
- Modify: `internal/backend/runtime_test.go`
- Modify: `app/catalogapi/interfaces.go`
- Modify: `app/catalogapi/server.go`
- Modify: `app/catalogapi/server_workflows.go`
- Modify: `app/catalogapi/catalog_read_module.go`
- Modify: `app/catalogapi/request_limiter.go`
- Modify: `app/catalogapi/common/params.go`
- Modify: `app/catalogapi/audit/query.go`
- Modify: `app/catalogapi/audit/query_test.go`
- Modify: `app/catalogapi/handlers/session/handlers.go`
- Modify: `app/catalogapi/server_test.go`
- Modify: `app/catalogapi/server_catalog_mutation_test.go`
- Modify: `app/catalogapi/server_workflow_seam_test.go`
- Modify: `app/catalogapi/server_packaging_access_test.go`
- Modify: `internal/catalog/http/binding.go`
- Modify: `internal/catalog/http/handlers.go`
- Modify: `internal/governance/http/routes.go`
- Modify: `internal/governance/http/routes_test.go`
- Modify: `internal/packaging/domain/types.go`
- Modify: `internal/packaging/domain/bundle_rules.go`
- Modify: `internal/packaging/application/service.go`
- Modify: `internal/packaging/application/publish_service.go`
- Modify: `internal/packaging/application/publish_service_test.go`
- Modify: `internal/packaging/http/workflow.go`
- Modify: `internal/packaging/http/handlers.go`
- Modify: `internal/packaging/http/binding.go`
- Modify: `internal/packaging/infra/workspace_adapter.go`
- Modify: `internal/packaging/infra/workspace_backing.go`
- Modify: `internal/packaging/infra/workspace_metadata.go`
- Modify: `internal/packaging/infra/workspace_pack_helpers.go`
- Modify: `internal/packaging/infra/workspace_source.go`
- Modify: `internal/platform/artifact/publisher.go`
- Modify: `internal/platform/artifact/sources.go`
- Modify: `internal/platform/artifact/sources_test.go`
- Modify: `internal/platform/auth/authorizer.go`
- Modify: `internal/platform/http/middleware/auth.go`
- Modify: `internal/platform/http/request_limiter.go`
- Modify: `internal/platform/http/response.go`
- Modify: `internal/platform/persistence/audit_store.go`
- Delete: `business/catalog/*.go`

- [ ] **Step 1: Write the failing retained-seam tests**

```go
// internal/backend/runtime_test.go
func TestNewRuntimeUsesInternalCatalogInfraRepository(t *testing.T) {
	root := t.TempDir()

	runtime, err := NewRuntime(context.Background(), Config{
		DatabaseURL:         "file:" + filepath.Join(root, "runtime.db"),
		SharedWorkspaceRoot: filepath.Join(root, "workspaces"),
		ArtifactSource:      cataloginfra.NewStaticArtifactSource(nil),
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	if _, ok := any(runtime.Repository).(*cataloginfra.Repository); !ok {
		t.Fatalf("expected internal catalog infra repository, got %T", runtime.Repository)
	}
}
```

```go
// cmd/skill-registry/main_test.go
func TestConfiguredArtifactSourceFallsBackToInternalStaticSource(t *testing.T) {
	t.Setenv("SKILL_REGISTRY_ARTIFACTS_PATH", "")
	t.Setenv("SKILL_REGISTRY_NPM_REGISTRY_URL", "")

	if _, ok := configuredArtifactSource().(*platformartifact.StaticArtifactSource); !ok {
		t.Fatalf("expected internal static artifact source, got %T", configuredArtifactSource())
	}
}
```

- [ ] **Step 2: Run the focused tests to verify they fail**

Run: `go test ./internal/backend ./cmd/skill-registry -run 'TestNewRuntimeUsesInternalCatalogInfraRepository|TestConfiguredArtifactSourceFallsBackToInternalStaticSource'`

Expected: FAIL because `cataloginfra.Repository` and `platformartifact.StaticArtifactSource` do not exist yet.

- [ ] **Step 3: Add the retained catalog constructors and migrate imports**

```go
// internal/catalog/application/provider.go
package application

type ServiceProvider interface {
	CatalogReadService() *ReadService
	CatalogWriteService() *WriteService
}
```

```go
// internal/catalog/infra/repository.go
package infra

type Repository struct {
	reads  *catalogapp.ReadService
	writes *catalogapp.WriteService
}

func NewRepository(config RepositoryConfig) *Repository { /* ... */ }
func NewMemoryRepository(data SampleCatalogData, options ...MemoryRepositoryOptions) *Repository { /* ... */ }
func NewSQLRepository(ctx context.Context, databaseURL string, seedData SampleCatalogData, options SQLRepositoryOptions) (*Repository, error) { /* ... */ }
func NewStaticArtifactSource(artifacts []ReleaseArtifact) *StaticArtifactSource { /* ... */ }
```

```go
// internal/backend/runtime.go
type Config struct {
	Addr                string
	DatabaseURL         string
	SharedWorkspaceRoot string
	ArtifactSource      catalogdomain.ArtifactSource
	Server              catalogapi.ServerConfig
	HTTPServer          catalogapi.HTTPServerConfig
	DBPool              DBPoolConfig
	IdempotencyTTL      time.Duration
}

repository := cataloginfra.NewRepository(cataloginfra.RepositoryConfig{
	StateStore:     internalStateStore,
	ArtifactSource: repositoryArtifactSource,
})
```

- [ ] **Step 4: Delete `business/catalog` after callers are migrated**

```bash
rm -r business/catalog
```

- [ ] **Step 5: Run the catalog-removal verification**

Run: `go test ./cmd/skill-registry ./internal/backend ./app/catalogapi ./internal/catalog/... ./internal/governance/http ./internal/packaging/... ./internal/platform/...`

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/skill-registry internal/catalog internal/backend app/catalogapi internal/governance internal/packaging internal/platform
git add -u business/catalog
git commit -m "refactor: remove business catalog compatibility layer"
```

### Task 2: Delete `business/packaging`

**Files:**
- Modify: `app/catalogapi/server.go`
- Modify: `app/catalogapi/server_workflows.go`
- Modify: `app/catalogapi/server_test.go`
- Modify: `app/catalogapi/server_workflow_seam_test.go`
- Modify: `app/catalogapi/server_packaging_access_test.go`
- Modify: `app/catalogapi/packaging_workflow_test.go`
- Modify: `app/catalogapi/module_test_helpers_test.go`
- Modify: `internal/packaging/http/workflow.go`
- Modify: `internal/packaging/http/handlers.go`
- Delete: `business/packaging/*.go`

- [ ] **Step 1: Write the failing packaging retained-seam test**

```go
// app/catalogapi/server_workflow_seam_test.go
func TestServerUsesInternalPackagingServiceTypes(t *testing.T) {
	server := NewServer(ServerConfig{})
	if server == nil {
		t.Fatal("expected server")
	}

	var _ packagingapplication.WorkflowService
}
```

- [ ] **Step 2: Run the focused test to verify it fails**

Run: `go test ./app/catalogapi -run TestServerUsesInternalPackagingServiceTypes`

Expected: FAIL while `app/catalogapi` and `internal/packaging/http` still import `business/packaging`.

- [ ] **Step 3: Migrate packaging callers to `internal/packaging/*`**

```go
// app/catalogapi/server.go
type packagingApplication interface {
	packagingapplication.WorkflowService
	packagingapplication.AccessService
	PublishContext(ctx context.Context, input packagingdomain.PublishRequest) (packagingdomain.PublishResponse, error)
	ConfigurePublishWorkflow(repo packagingapplication.ReleaseRecordCreator, publisher packagingapplication.PackagePublisher)
}
```

```go
// internal/packaging/http/workflow.go
func (w *Workflow) Prepare(ctx context.Context, principal auth.Principal, payload packagingdomain.PrepareRequest) (any, int, *catalog.Error) { /* ... */ }
```

- [ ] **Step 4: Delete `business/packaging`**

```bash
rm -r business/packaging
```

- [ ] **Step 5: Run the packaging-removal verification**

Run: `go test ./app/catalogapi ./internal/packaging/...`

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add app/catalogapi internal/packaging
git add -u business/packaging
git commit -m "refactor: remove business packaging compatibility layer"
```

### Task 3: Shrink `app/catalogapi` to retained owners

**Files:**
- Modify: `app/catalogapi/server.go`
- Modify: `app/catalogapi/interfaces.go`
- Modify: `internal/catalog/http/handlers.go`
- Modify: `internal/governance/http/routes.go`
- Modify: `internal/system/http/routes.go`

- [ ] **Step 1: Write the failing system-owner test**

```go
// app/catalogapi/server_workflow_seam_test.go
func TestServerUsesInternalSystemHTTPTypes(t *testing.T) {
	var _ internalsystemhttp.Handlers
}
```

- [ ] **Step 2: Run the focused test to verify it fails**

Run: `go test ./app/catalogapi ./internal/system/http -run TestServerUsesInternalSystemHTTPTypes`

Expected: FAIL while `app/catalogapi/interfaces.go` still depends on `app/catalogapi/system`.

- [ ] **Step 3: Move remaining interface seams to retained owners**

```go
// app/catalogapi/interfaces.go
type systemWorkflow interface {
	Readyz(ctx context.Context) (internalsystemhttp.Readyz, int)
}
```

- [ ] **Step 4: Run the end-state verification**

Run: `go test ./...`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/catalogapi internal/catalog/http internal/governance/http internal/system/http
git commit -m "refactor: shrink catalogapi owners to retained seams"
```
