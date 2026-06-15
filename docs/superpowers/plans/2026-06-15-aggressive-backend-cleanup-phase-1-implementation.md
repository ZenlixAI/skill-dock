# Aggressive Backend Cleanup Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete the shallow backend facade modules, migrate in-repo imports to the retained `internal/*` seams, and keep the HTTP product contract stable.

**Architecture:** Implement the cleanup in three mechanical batches: shared platform facades first, `app/catalogapi` binding/handler facades second, and alias-only `business/*` vocabulary last. Each batch moves callers onto the retained seam before deleting the old module, so behavior stays stable while the file graph shrinks.

**Tech Stack:** Go 1.26, Gin, standard library `net/http`, existing Go test suite

---

## File Structure

### Batch 1: Shared platform seam cleanup

- Modify: `cmd/skill-registry/main.go`
- Modify: `cmd/skill-registry/main_test.go`
- Modify: `internal/backend/runtime.go`
- Modify: `internal/backend/runtime_test.go`
- Modify: `app/catalogapi/server.go`
- Modify: `app/catalogapi/interfaces.go`
- Modify: `app/catalogapi/access_workflow.go`
- Modify: `app/catalogapi/audit/query.go`
- Modify: `app/catalogapi/audit/recorder.go`
- Modify: `app/catalogapi/session_module.go`
- Modify: `internal/catalog/http/handlers.go`
- Modify: `internal/governance/http/routes.go`
- Modify: `internal/governance/http/routes_test.go`
- Modify: `internal/packaging/http/handlers.go`
- Modify: `internal/packaging/http/workflow.go`
- Modify: `internal/system/http/routes.go`
- Modify: `internal/packaging/application/publish_service.go`
- Modify: `internal/packaging/application/publish_service_test.go`
- Modify: `internal/packaging/application/service.go`
- Modify: `internal/packaging/application/workspace_service.go`
- Modify: `internal/packaging/application/workspace_service_test.go`
- Modify: `internal/packaging/infra/workspace_adapter.go`
- Modify: `internal/packaging/infra/workspace_backing.go`
- Modify: `internal/packaging/infra/workspace_pack_helpers.go`
- Delete: `app/auth/authorizer.go`
- Delete: `app/auth/context.go`
- Delete: `app/auth/session_manager.go`
- Delete: `app/observability/http.go`
- Delete: `app/observability/metrics.go`
- Delete: `foundation/artifact/publisher.go`
- Delete: `foundation/artifact/scan.go`
- Delete: `foundation/artifact/sources.go`
- Delete: `foundation/persist/audit_store.go`
- Delete: `foundation/persist/idempotency_store.go`
- Delete: `foundation/persist/store_set.go`
- Delete: `foundation/persist/workspace_store.go`
- Delete: `app/auth/platform_alias_test.go`
- Delete: `app/observability/platform_alias_test.go`
- Delete: `foundation/artifact/platform_alias_test.go`
- Delete: `foundation/persist/platform_alias_test.go`

### Batch 2: `app/catalogapi` binding and handler facade cleanup

- Modify: `app/catalogapi/server.go`
- Modify: `app/catalogapi/catalog_read_module.go`
- Modify: `app/catalogapi/module_test_helpers_test.go`
- Modify: `app/catalogapi/server_test.go`
- Delete: `app/catalogapi/binding/catalog.go`
- Delete: `app/catalogapi/binding/packaging.go`
- Delete: `app/catalogapi/handlers/catalog/handlers.go`
- Delete: `app/catalogapi/handlers/governance/handlers.go`
- Delete: `app/catalogapi/handlers/packaging/handlers.go`
- Delete: `app/catalogapi/handlers/system/handlers.go`

### Batch 3: Alias-only `business/*` vocabulary cleanup

- Modify: `cmd/skill-registry/main.go`
- Modify: `internal/backend/runtime.go`
- Modify: `app/catalogapi/server.go`
- Modify: `app/catalogapi/server_workflows.go`
- Modify: `app/catalogapi/catalog_read_module.go`
- Modify: `app/catalogapi/common/params.go`
- Modify: `app/catalogapi/request_limiter.go`
- Modify: `app/catalogapi/audit/query.go`
- Modify: `app/catalogapi/handlers/session/handlers.go`
- Modify: `internal/catalog/http/binding.go`
- Modify: `internal/catalog/http/handlers.go`
- Modify: `internal/governance/http/routes.go`
- Modify: `internal/packaging/http/binding.go`
- Modify: `internal/packaging/http/handlers.go`
- Modify: `internal/packaging/http/workflow.go`
- Modify: `internal/packaging/application/publish_service.go`
- Modify: `internal/packaging/application/service.go`
- Modify: `internal/packaging/domain/bundle_rules.go`
- Modify: `internal/packaging/domain/types.go`
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
- Delete: `business/catalog/types.go`
- Delete: `business/catalog/errors.go`
- Delete: `business/catalog/repository.go`
- Delete: `business/packaging/service.go`
- Delete: `business/packaging/types.go`

### End-state verification

- Test: `cmd/skill-registry/main_test.go`
- Test: `internal/backend/runtime_test.go`
- Test: `app/catalogapi/server_test.go`
- Test: `app/catalogapi/server_catalog_mutation_test.go`
- Test: `app/catalogapi/server_packaging_access_test.go`
- Test: `app/catalogapi/server_workflow_seam_test.go`
- Test: `app/catalogapi/packaging_workflow_test.go`
- Test: `app/catalogapi/access_workflow_test.go`
- Test: `app/catalogapi/audit/query_test.go`
- Test: `app/catalogapi/audit/recorder_test.go`
- Test: `business/catalog/read_module_test.go`
- Test: `business/catalog/write_module_test.go`
- Test: `business/catalog/sql_repository_test.go`
- Test: `business/packaging/published_release_workflow_test.go`
- Test: `business/packaging/workspace_runtime_module_test.go`
- Test: `business/packaging/workspace_source_module_test.go`
- Test: `internal/catalog/http/routes_test.go`
- Test: `internal/governance/http/routes_test.go`
- Test: `internal/packaging/application/*.go`
- Test: `internal/packaging/http/routes_test.go`
- Test: `internal/packaging/infra/workspace_source_test.go`
- Test: `internal/platform/artifact/*.go`
- Test: `internal/platform/http/*.go`
- Test: `internal/platform/persistence/*.go`

### Task 1: Remove shared platform facade modules

**Files:**
- Modify: `cmd/skill-registry/main.go`
- Modify: `internal/backend/runtime.go`
- Modify: `app/catalogapi/server.go`
- Modify: `app/catalogapi/interfaces.go`
- Modify: `app/catalogapi/access_workflow.go`
- Modify: `app/catalogapi/audit/query.go`
- Modify: `app/catalogapi/audit/recorder.go`
- Modify: `app/catalogapi/session_module.go`
- Modify: `internal/catalog/http/handlers.go`
- Modify: `internal/governance/http/routes.go`
- Modify: `internal/packaging/http/handlers.go`
- Modify: `internal/packaging/http/workflow.go`
- Modify: `internal/system/http/routes.go`
- Modify: `internal/packaging/application/publish_service.go`
- Modify: `internal/packaging/application/service.go`
- Modify: `internal/packaging/application/workspace_service.go`
- Modify: `internal/packaging/infra/workspace_adapter.go`
- Modify: `internal/packaging/infra/workspace_backing.go`
- Modify: `internal/packaging/infra/workspace_pack_helpers.go`
- Delete: `app/auth/*.go`
- Delete: `app/observability/*.go`
- Delete: `foundation/artifact/*.go`
- Delete: `foundation/persist/*.go`

- [ ] **Step 1: Write the failing seam-ownership tests**

```go
// cmd/skill-registry/main_test.go
package main

import (
	"testing"

	platformauth "skill_registry/internal/platform/auth"
	platformartifact "skill_registry/internal/platform/artifact"
)

func TestConfiguredAuthTokensUsePlatformTypes(t *testing.T) {
	t.Setenv("SKILL_REGISTRY_AUTH_TOKENS_JSON", `{"reader-token":{"role":"reader","actor":"alice"}}`)
	tokens := configuredAuthTokens()
	if _, ok := tokens["reader-token"]; !ok {
		t.Fatal("expected configured token")
	}
	var _ platformauth.TokenConfig = tokens["reader-token"]
}

func TestConfiguredArtifactSourceReturnsPlatformAdapter(t *testing.T) {
	t.Setenv("SKILL_REGISTRY_NPM_REGISTRY_URL", "https://registry.example.com")
	if _, ok := configuredArtifactSource().(*platformartifact.NPMRegistryArtifactSource); !ok {
		t.Fatalf("expected internal platform artifact adapter, got %T", configuredArtifactSource())
	}
}
```

```go
// internal/backend/runtime_test.go
package backend

import (
	"testing"

	packagingapplication "skill_registry/internal/packaging/application"
)

func TestNewPackagingServiceReturnsInternalApplicationService(t *testing.T) {
	service := newPackagingService(t.TempDir(), nil)
	if service == nil {
		t.Fatal("expected packaging service")
	}
	var _ *packagingapplication.Service = service
}
```

- [ ] **Step 2: Run the focused tests to verify they fail**

Run: `go test ./cmd/skill-registry ./internal/backend -run 'TestConfiguredAuthTokensUsePlatformTypes|TestConfiguredArtifactSourceReturnsPlatformAdapter|TestRuntimeExposesInternalSeams'`

Expected: FAIL with undefined `newPackagingService` or missing internal imports before the import migration is complete.

- [ ] **Step 3: Migrate callers to internal platform seams**

```go
// cmd/skill-registry/main.go
import (
	// ...
	platformauth "skill_registry/internal/platform/auth"
	platformartifact "skill_registry/internal/platform/artifact"
	"skill_registry/internal/backend"
)

func configuredArtifactSource() catalog.ArtifactSource {
	if path := os.Getenv("SKILL_REGISTRY_ARTIFACTS_PATH"); path != "" {
		return platformartifact.NewJSONArtifactSource(path)
	}
	if registryURL := os.Getenv("SKILL_REGISTRY_NPM_REGISTRY_URL"); registryURL != "" {
		return platformartifact.NewNPMRegistryArtifactSource(registryURL, nil)
	}
	return catalog.NewStaticArtifactSource(nil)
}

func configuredAuthTokens() map[string]platformauth.TokenConfig {
	raw := os.Getenv("SKILL_REGISTRY_AUTH_TOKENS_JSON")
	if raw == "" {
		return nil
	}

	var authTokens map[string]platformauth.TokenConfig
	if err := json.Unmarshal([]byte(raw), &authTokens); err != nil {
		log.Fatalf("parse SKILL_REGISTRY_AUTH_TOKENS_JSON: %v", err)
	}
	return authTokens
}

func configuredSessionUsers() map[string]platformauth.SessionUserConfig {
	raw := os.Getenv("SKILL_REGISTRY_SESSION_USERS_JSON")
	if raw == "" {
		return nil
	}

	var sessionUsers map[string]platformauth.SessionUserConfig
	if err := json.Unmarshal([]byte(raw), &sessionUsers); err != nil {
		log.Fatalf("parse SKILL_REGISTRY_SESSION_USERS_JSON: %v", err)
	}
	return sessionUsers
}
```

```go
// internal/backend/runtime.go
import (
	// ...
	platformartifact "skill_registry/internal/platform/artifact"
	platformpersist "skill_registry/internal/platform/persistence"
	packagingapplication "skill_registry/internal/packaging/application"
)

type Runtime struct {
	DB           *sql.DB
	Repository   catalog.Repository
	StoreSet     *platformpersist.SQLStoreSet
	CatalogRead  *catalogapp.ReadService
	CatalogWrite *catalogapp.WriteService
	Server       *catalogapi.Server
	Handler      http.Handler
	HTTPServer   *http.Server
}

func newPackagingService(root string, store platformpersist.WorkspaceStore) *packagingapplication.Service {
	return packagingapplication.NewServiceWithWorkspaceStore(root, store)
}
```

```go
// app/catalogapi/server.go
import (
	// ...
	platformauth "skill_registry/internal/platform/auth"
	platformpersist "skill_registry/internal/platform/persistence"
	platformobservability "skill_registry/internal/platform/observability"
)

type ServerConfig struct {
	// ...
	AuthTokens   map[string]platformauth.TokenConfig
	SessionUsers map[string]platformauth.SessionUserConfig
	Idempotency  platformpersist.IdempotencyStore
	Audit        platformpersist.AuditStore
}
```

- [ ] **Step 4: Delete the platform facade modules and alias-only tests**

```bash
rm app/auth/authorizer.go app/auth/context.go app/auth/session_manager.go
rm app/observability/http.go app/observability/metrics.go
rm foundation/artifact/publisher.go foundation/artifact/scan.go foundation/artifact/sources.go
rm foundation/persist/audit_store.go foundation/persist/idempotency_store.go foundation/persist/store_set.go foundation/persist/workspace_store.go
rm app/auth/platform_alias_test.go app/observability/platform_alias_test.go foundation/artifact/platform_alias_test.go foundation/persist/platform_alias_test.go
```

- [ ] **Step 5: Run the batch-1 verification**

Run: `go test ./cmd/skill-registry ./internal/backend ./app/catalogapi ./internal/platform/... ./internal/catalog/http ./internal/governance/http ./internal/packaging/http`

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/skill-registry/main.go cmd/skill-registry/main_test.go internal/backend/runtime.go internal/backend/runtime_test.go app/catalogapi internal/catalog/http/handlers.go internal/governance/http/routes.go internal/packaging/http/handlers.go internal/packaging/http/workflow.go internal/system/http/routes.go internal/packaging/application internal/packaging/infra internal/platform
git add -u app/auth app/observability foundation/artifact foundation/persist
git commit -m "refactor: remove shared platform facades"
```

### Task 2: Remove `app/catalogapi` binding and handler facades

**Files:**
- Modify: `app/catalogapi/server.go`
- Modify: `app/catalogapi/catalog_read_module.go`
- Modify: `app/catalogapi/module_test_helpers_test.go`
- Modify: `app/catalogapi/server_test.go`
- Delete: `app/catalogapi/binding/catalog.go`
- Delete: `app/catalogapi/binding/packaging.go`
- Delete: `app/catalogapi/handlers/catalog/handlers.go`
- Delete: `app/catalogapi/handlers/governance/handlers.go`
- Delete: `app/catalogapi/handlers/packaging/handlers.go`
- Delete: `app/catalogapi/handlers/system/handlers.go`

- [ ] **Step 1: Write the failing direct-http seam tests**

```go
// app/catalogapi/server_test.go
package catalogapi

import (
	"testing"

	internalcataloghttp "skill_registry/internal/catalog/http"
	internalgovernancehttp "skill_registry/internal/governance/http"
	internalpackaginghttp "skill_registry/internal/packaging/http"
	internalsystemhttp "skill_registry/internal/system/http"
)

func TestServerUsesInternalHTTPHandlers(t *testing.T) {
	server := NewServer(ServerConfig{})
	if server == nil {
		t.Fatal("expected server")
	}
	var _ internalcataloghttp.Handlers
	var _ internalgovernancehttp.Handlers
	var _ internalpackaginghttp.Handlers
	var _ internalsystemhttp.Handlers
}
```

- [ ] **Step 2: Run the focused tests to verify they fail**

Run: `go test ./app/catalogapi -run TestServerUsesInternalHTTPHandlers`

Expected: FAIL while `app/catalogapi/server.go` still imports the facade handler packages.

- [ ] **Step 3: Switch server wiring to internal handler and binding packages directly**

```go
// app/catalogapi/server.go
import (
	// ...
	internalcataloghttp "skill_registry/internal/catalog/http"
	internalgovernancehttp "skill_registry/internal/governance/http"
	internalpackaginghttp "skill_registry/internal/packaging/http"
	internalsystemhttp "skill_registry/internal/system/http"
)

catalogHTTPHandlers := internalcataloghttp.New(internalcataloghttp.Handlers{
	AuthorizePackage:     s.authorizePackageAccess,
	ServeCachedReadJSON:  s.serveCachedReadJSON,
	ReadListCacheTTL:     s.readListCacheTTL,
	ReadDetailCacheTTL:   s.readDetailCacheTTL,
	ListAuditEvents:      s.auditQueryModule.ListEvents,
	ListSkillsFunc:       s.catalogReadModule.listSkills,
	ListCuratedSkills:    s.catalogReadModule.listCuratedSkills,
	GetSkillFunc:         s.catalogReadModule.getSkill,
	ListVersionsFunc:     s.catalogReadModule.listVersions,
	GetVersionFunc:       s.catalogReadModule.getVersion,
	ResolveInstallTarget: s.catalogReadModule.resolveInstallTarget,
	PublicInstallTarget:  s.publicInstallTarget,
})
governanceHTTPHandlers := internalgovernancehttp.New(internalgovernancehttp.Handlers{
	ServeCachedReadJSON: s.serveCachedReadJSON,
	ReadListCacheTTL:    s.readListCacheTTL,
	AuthorizePackage:    s.authorizePackageAccess,
	AuthorizeAction:     s.accessWorkflow.authorizeAction,
	ListGovernance:      s.listGovernanceChangesFunc,
	AcquireWriteLimiter: func() (func(), *catalog.Error) { return s.writeLimiter.acquire() },
	CreateReleaseRecord: func(request *http.Request, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error) {
		return s.createReleaseRecordFunc(request.Context(), principal, packageName, versionPath, rawBody)
	},
	ApplySkillGovernanceChange: func(request *http.Request, principal auth.Principal, packageName string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
		return s.applySkillGovernanceChangeFunc(request.Context(), principal, packageName, rawBody)
	},
	ApplyVersionGovernance: func(request *http.Request, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
		return s.applyVersionGovernanceChangeFunc(request.Context(), principal, packageName, versionPath, rawBody)
	},
})
packagingWorkflow := internalpackaginghttp.NewWorkflow(internalpackaginghttp.WorkflowConfig{
	Service:             packagingSvc,
	Idempotency:         config.Idempotency,
	Audit:               auditRecorder,
	CacheTTL:            10 * time.Minute,
	AllowAnonymousCache: false,
})
systemHandlers := internalsystemhttp.New(server.systemModule.Readyz, server.metrics.RenderPrometheus)
```

```go
// app/catalogapi/catalog_read_module.go
import internalcataloghttp "skill_registry/internal/catalog/http"

type catalogReadWorkflow interface {
	listSkills(ctx context.Context, principal auth.Principal, query internalcataloghttp.SkillsQuery) (catalog.SkillsPage, *catalog.Error)
	listCuratedSkills(ctx context.Context, principal auth.Principal, pageSize int, cursor int) (catalog.CuratedSkillsPage, *catalog.Error)
	getSkill(ctx context.Context, packageName string) (catalog.SkillDetail, error)
	listVersions(ctx context.Context, query catalog.VersionQuery) (catalog.VersionsPage, error)
	getVersion(ctx context.Context, packageName string, version string) (catalog.SkillVersionDetail, error)
	resolveInstallTarget(ctx context.Context, packageName string, version string) (catalog.InstallResolution, error)
}
```

- [ ] **Step 4: Delete the facade binding and handler files**

```bash
rm app/catalogapi/binding/catalog.go app/catalogapi/binding/packaging.go
rm app/catalogapi/handlers/catalog/handlers.go
rm app/catalogapi/handlers/governance/handlers.go
rm app/catalogapi/handlers/packaging/handlers.go
rm app/catalogapi/handlers/system/handlers.go
```

- [ ] **Step 5: Run the batch-2 verification**

Run: `go test ./app/catalogapi ./internal/catalog/http ./internal/governance/http ./internal/packaging/http ./internal/system/http`

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add app/catalogapi/server.go app/catalogapi/catalog_read_module.go app/catalogapi/module_test_helpers_test.go app/catalogapi/server_test.go
git add -u app/catalogapi/binding app/catalogapi/handlers/catalog app/catalogapi/handlers/governance app/catalogapi/handlers/packaging app/catalogapi/handlers/system
git commit -m "refactor: drop catalogapi handler facades"
```

### Task 3: Remove alias-only `business/*` vocabulary files

**Files:**
- Modify: `cmd/skill-registry/main.go`
- Modify: `internal/backend/runtime.go`
- Modify: `app/catalogapi/server.go`
- Modify: `app/catalogapi/server_workflows.go`
- Modify: `app/catalogapi/catalog_read_module.go`
- Modify: `app/catalogapi/common/params.go`
- Modify: `app/catalogapi/request_limiter.go`
- Modify: `app/catalogapi/audit/query.go`
- Modify: `app/catalogapi/handlers/session/handlers.go`
- Modify: `internal/catalog/http/binding.go`
- Modify: `internal/catalog/http/handlers.go`
- Modify: `internal/governance/http/routes.go`
- Modify: `internal/packaging/http/binding.go`
- Modify: `internal/packaging/http/handlers.go`
- Modify: `internal/packaging/http/workflow.go`
- Modify: `internal/packaging/application/publish_service.go`
- Modify: `internal/packaging/application/service.go`
- Modify: `internal/packaging/domain/bundle_rules.go`
- Modify: `internal/packaging/domain/types.go`
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
- Delete: `business/catalog/types.go`
- Delete: `business/catalog/errors.go`
- Delete: `business/catalog/repository.go`
- Delete: `business/packaging/service.go`
- Delete: `business/packaging/types.go`

- [ ] **Step 1: Write the failing direct-domain seam tests**

```go
// internal/catalog/http/routes_test.go
package http

import (
	"testing"

	internaldomain "skill_registry/internal/catalog/domain"
)

func TestCatalogHTTPUsesInternalDomainVocabulary(t *testing.T) {
	var query SkillsQuery
	_ = query
	var _ internaldomain.SkillQuery
}
```

```go
// internal/packaging/http/routes_test.go
package http

import (
	"testing"

	packagingdomain "skill_registry/internal/packaging/domain"
)

func TestPackagingHTTPUsesInternalDomainVocabulary(t *testing.T) {
	var _ packagingdomain.PublishRequest
	var _ packagingdomain.PackRequest
}
```

- [ ] **Step 2: Run the focused tests to verify they fail**

Run: `go test ./internal/catalog/http ./internal/packaging/http -run 'TestCatalogHTTPUsesInternalDomainVocabulary|TestPackagingHTTPUsesInternalDomainVocabulary'`

Expected: FAIL while the HTTP packages and remaining callers still depend on `business/catalog` and `business/packaging` alias vocabulary.

- [ ] **Step 3: Migrate imports to internal domain/application seams**

```go
// internal/catalog/http/binding.go
import internaldomain "skill_registry/internal/catalog/domain"

func BindSkillsQuery(context *gin.Context) (SkillsQuery, *internaldomain.Error) {
	return bindQuery[SkillsQuery](context)
}

func DecodePackageName(raw string) (string, *internaldomain.Error) {
	packageName, err := url.PathUnescape(raw)
	if err != nil {
		return "", internaldomain.NewError(400, "invalid_request", "package path is not valid URL encoding.", map[string]any{"package": raw})
	}
	return packageName, nil
}
```

```go
// internal/packaging/http/binding.go
import (
	// ...
	internaldomain "skill_registry/internal/catalog/domain"
	packagingdomain "skill_registry/internal/packaging/domain"
)

type BundleDownloadQuery struct {
	WorkspacePath string `form:"workspacePath"`
	PackageName   string `form:"packageName"`
	Version       string `form:"version"`
}

func BindBundleDownloadQuery(context *gin.Context) (BundleDownloadQuery, *internaldomain.Error) {
	var query BundleDownloadQuery
	if err := context.ShouldBindQuery(&query); err != nil {
		return query, internaldomain.NewError(400, "invalid_request", "query parameters are invalid.", map[string]any{"error": err.Error()})
	}
	return query, nil
}
```

```go
// internal/packaging/http/handlers.go
import (
	// ...
	internaldomain "skill_registry/internal/catalog/domain"
	packagingdomain "skill_registry/internal/packaging/domain"
)

type DecodePrepareRequestFunc func(context.Context, platformauth.Principal, []byte) (packagingdomain.PrepareRequest, *internaldomain.Error)
type PublishFunc func(context.Context, packagingdomain.PublishRequest) (packagingdomain.PublishResponse, error)
```

```go
// app/catalogapi/server.go
import (
	// ...
	internaldomain "skill_registry/internal/catalog/domain"
	packagingdomain "skill_registry/internal/packaging/domain"
)

type catalogMutationService interface {
	CreateReleaseRecord(ctx context.Context, packageName string, version string, input internaldomain.ReleaseRecordInput) (internaldomain.ReleaseRecordResult, error)
}

func (s *Server) cleanupPackaging(ctx context.Context, principal platformauth.Principal, payload packagingdomain.CleanupRequest) (packagingdomain.CleanupResponse, int, *internaldomain.Error) {
	if strings.TrimSpace(payload.WorkspacePath) == "" {
		err := internaldomain.NewError(http.StatusUnprocessableEntity, "invalid_request", "workspacePath is required.", nil)
		s.recordMutationAuditForEndpoint(ctx, principal, http.MethodPost, "/api/v1/skill-packaging/cleanup", "packaging.cleanup", "", "", err.Status, "failed", map[string]any{"errorCode": err.Code})
		return packagingdomain.CleanupResponse{}, 0, err
	}
	if s.packaging == nil {
		return packagingdomain.CleanupResponse{}, http.StatusNotFound, nil
	}
	if err := s.packaging.CleanupContext(ctx, payload.WorkspacePath); err != nil {
		catalogErr := common.CatalogErrorFromError(err)
		s.recordMutationAuditForEndpoint(ctx, principal, http.MethodPost, "/api/v1/skill-packaging/cleanup", "packaging.cleanup", "", "", catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
		return packagingdomain.CleanupResponse{}, 0, catalogErr
	}
	s.recordMutationAuditForEndpoint(ctx, principal, http.MethodPost, "/api/v1/skill-packaging/cleanup", "packaging.cleanup", "", "", http.StatusOK, "succeeded", map[string]any{"workspacePath": payload.WorkspacePath})
	return packagingdomain.CleanupResponse{WorkspacePath: payload.WorkspacePath, Removed: true}, http.StatusOK, nil
}
```

- [ ] **Step 4: Delete the alias-only vocabulary files**

```bash
rm business/catalog/types.go business/catalog/errors.go business/catalog/repository.go
rm business/packaging/service.go business/packaging/types.go
```

- [ ] **Step 5: Run the batch-3 verification**

Run: `go test ./app/catalogapi ./business/catalog ./business/packaging ./internal/... ./cmd/skill-registry`

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/skill-registry/main.go internal/backend/runtime.go app/catalogapi app/catalogapi/common internal/catalog/http internal/governance/http internal/packaging internal/platform
git add -u business/catalog business/packaging
git commit -m "refactor: remove alias-only business vocabulary"
```

### Task 4: Final verification and cleanup sweep

**Files:**
- Modify: any remaining imports or tests touched by Tasks 1-3
- Test: `go test ./...`

- [ ] **Step 1: Run the full backend verification suite**

Run: `go test ./...`

Expected: PASS

- [ ] **Step 2: Confirm no deleted facade imports remain**

Run: `rg -n 'skill_registry/app/auth|skill_registry/app/observability|skill_registry/foundation/artifact|skill_registry/foundation/persist|skill_registry/app/catalogapi/binding|skill_registry/app/catalogapi/handlers/catalog|skill_registry/app/catalogapi/handlers/governance|skill_registry/app/catalogapi/handlers/packaging|skill_registry/app/catalogapi/handlers/system|skill_registry/business/catalog|skill_registry/business/packaging' app business foundation internal cmd`

Expected: only the intentionally retained behavior-owning `business/catalog/*`, `business/packaging/*`, and `app/catalogapi/handlers/session/handlers.go` references remain.

- [ ] **Step 3: Commit the final cleanup sweep**

```bash
git add -A
git commit -m "refactor: finish aggressive backend cleanup phase 1"
```
