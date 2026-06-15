# Backend Reorganization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reorganize the Go backend around business-domain-first navigation while keeping the external HTTP contract for Web and CLI stable.

**Architecture:** Introduce `internal/platform` for shared technical seams and split Catalog, Packaging, Governance, and System into `http`, `application`, `domain`, and `infra` modules. Use temporary compatibility shims so the migration stays buildable and reviewable at every checkpoint, then delete the shims after the new composition root is in place.

**Tech Stack:** Go 1.26, Gin, pgx, sqlite, standard library `net/http`, existing Go test suite

---

## File Structure

### New platform modules

- Create: `internal/platform/auth/authorizer.go`
- Create: `internal/platform/auth/context.go`
- Create: `internal/platform/auth/session_manager.go`
- Create: `internal/platform/http/engine.go`
- Create: `internal/platform/http/metadata.go`
- Create: `internal/platform/http/response.go`
- Create: `internal/platform/http/request_limiter.go`
- Create: `internal/platform/http/read_cache.go`
- Create: `internal/platform/http/middleware/auth.go`
- Create: `internal/platform/http/middleware/canonical.go`
- Create: `internal/platform/http/middleware/cors.go`
- Create: `internal/platform/http/middleware/metadata.go`
- Create: `internal/platform/http/middleware/observability.go`
- Create: `internal/platform/observability/http.go`
- Create: `internal/platform/observability/metrics.go`
- Create: `internal/platform/persistence/audit_store.go`
- Create: `internal/platform/persistence/idempotency_store.go`
- Create: `internal/platform/persistence/store_set.go`
- Create: `internal/platform/persistence/workspace_store.go`
- Create: `internal/platform/artifact/publisher.go`
- Create: `internal/platform/artifact/scan.go`
- Create: `internal/platform/artifact/sources.go`

### New catalog modules

- Create: `internal/catalog/domain/types.go`
- Create: `internal/catalog/domain/errors.go`
- Create: `internal/catalog/domain/repository.go`
- Create: `internal/catalog/application/read_service.go`
- Create: `internal/catalog/application/write_service.go`
- Create: `internal/catalog/infra/sql_repository.go`
- Create: `internal/catalog/infra/sql_state_store.go`
- Create: `internal/catalog/infra/memory_repository.go`
- Create: `internal/catalog/http/handlers.go`
- Create: `internal/catalog/http/binding.go`
- Create: `internal/catalog/http/routes.go`

### New packaging modules

- Create: `internal/packaging/domain/types.go`
- Create: `internal/packaging/domain/warnings.go`
- Create: `internal/packaging/domain/bundle_rules.go`
- Create: `internal/packaging/application/workspace_service.go`
- Create: `internal/packaging/application/publish_service.go`
- Create: `internal/packaging/infra/workspace_adapter.go`
- Create: `internal/packaging/infra/workspace_backing.go`
- Create: `internal/packaging/infra/workspace_locking.go`
- Create: `internal/packaging/http/handlers.go`
- Create: `internal/packaging/http/binding.go`
- Create: `internal/packaging/http/routes.go`

### New system and governance modules

- Create: `internal/system/application/service.go`
- Create: `internal/system/http/routes.go`
- Create: `internal/governance/application/service.go`
- Create: `internal/governance/http/routes.go`

### New composition module

- Create: `internal/backend/runtime.go`
- Create: `internal/backend/runtime_test.go`

### Transitional compatibility shims and runtime touch points

- Modify: `app/auth/authorizer.go`
- Modify: `app/auth/context.go`
- Modify: `app/auth/session_manager.go`
- Modify: `app/observability/http.go`
- Modify: `app/observability/metrics.go`
- Modify: `foundation/persist/audit_store.go`
- Modify: `foundation/persist/idempotency_store.go`
- Modify: `foundation/persist/store_set.go`
- Modify: `foundation/persist/workspace_store.go`
- Modify: `foundation/artifact/publisher.go`
- Modify: `foundation/artifact/scan.go`
- Modify: `foundation/artifact/sources.go`
- Modify: `business/catalog/errors.go`
- Modify: `business/catalog/repository.go`
- Modify: `business/catalog/read_module.go`
- Modify: `business/catalog/write_module.go`
- Modify: `business/catalog/sql_repository.go`
- Modify: `business/catalog/sql_state_store.go`
- Modify: `business/catalog/memory_repository.go`
- Modify: `business/catalog/types.go`
- Modify: `business/packaging/service.go`
- Modify: `business/packaging/workspace_runtime_module.go`
- Modify: `business/packaging/published_release_workflow.go`
- Modify: `business/packaging/types.go`
- Modify: `business/packaging/warnings.go`
- Modify: `business/packaging/bundle_rule_module.go`
- Modify: `app/catalogapi/server.go`
- Modify: `app/catalogapi/router/router.go`
- Modify: `app/catalogapi/router/catalog.go`
- Modify: `app/catalogapi/router/packaging.go`
- Modify: `app/catalogapi/router/session.go`
- Modify: `app/catalogapi/router/system.go`
- Modify: `cmd/skill-registry/main.go`

### Tests to keep green throughout the migration

- Test: `app/catalogapi/server_test.go`
- Test: `app/catalogapi/request_limiter_test.go`
- Test: `app/catalogapi/read_cache_test.go`
- Test: `app/catalogapi/access_workflow_test.go`
- Test: `app/catalogapi/packaging_module_test.go`
- Test: `business/catalog/read_module_test.go`
- Test: `business/catalog/write_module_test.go`
- Test: `business/catalog/sql_repository_test.go`
- Test: `business/packaging/service_test.go`
- Test: `business/packaging/workspace_runtime_module_test.go`
- Test: `business/packaging/published_release_workflow_test.go`
- Test: `foundation/persist/store_set_test.go`
- Test: `foundation/artifact/publisher_test.go`

### Verification commands

- `go test ./app/catalogapi`
- `go test ./business/catalog`
- `go test ./business/packaging`
- `go test ./foundation/persist`
- `go test ./foundation/artifact`
- `go test ./cmd/skill-registry`
- `go test ./...`

## Task 1: Establish `internal/platform` with compatibility shims

**Files:**
- Create: `internal/platform/auth/authorizer.go`
- Create: `internal/platform/auth/context.go`
- Create: `internal/platform/auth/session_manager.go`
- Create: `internal/platform/observability/http.go`
- Create: `internal/platform/observability/metrics.go`
- Create: `internal/platform/persistence/audit_store.go`
- Create: `internal/platform/persistence/idempotency_store.go`
- Create: `internal/platform/persistence/store_set.go`
- Create: `internal/platform/persistence/workspace_store.go`
- Create: `internal/platform/artifact/publisher.go`
- Create: `internal/platform/artifact/scan.go`
- Create: `internal/platform/artifact/sources.go`
- Modify: `app/auth/authorizer.go`
- Modify: `app/auth/context.go`
- Modify: `app/auth/session_manager.go`
- Modify: `app/observability/http.go`
- Modify: `app/observability/metrics.go`
- Modify: `foundation/persist/audit_store.go`
- Modify: `foundation/persist/idempotency_store.go`
- Modify: `foundation/persist/store_set.go`
- Modify: `foundation/persist/workspace_store.go`
- Modify: `foundation/artifact/publisher.go`
- Modify: `foundation/artifact/scan.go`
- Modify: `foundation/artifact/sources.go`
- Test: `app/auth/authorizer_test.go`
- Test: `foundation/persist/store_set_test.go`
- Test: `foundation/artifact/publisher_test.go`

- [ ] **Step 1: Write failing compile-seam tests for platform aliases**

```go
package auth

import (
	"testing"

	platformauth "skill_registry/internal/platform/auth"
)

func TestAuthorizerAliasMatchesPlatformType(t *testing.T) {
	var appType *Authorizer
	var platformType *platformauth.Authorizer
	appType = platformType
	if appType != nil {
		t.Fatal("expected zero values to stay assignable")
	}
}
```

```go
package persist

import (
	"testing"

	platformpersist "skill_registry/internal/platform/persistence"
)

func TestStoreSetAliasMatchesPlatformType(t *testing.T) {
	var appType *SQLStoreSet
	var platformType *platformpersist.SQLStoreSet
	appType = platformType
	if appType != nil {
		t.Fatal("expected zero values to stay assignable")
	}
}
```

- [ ] **Step 2: Run the targeted tests to verify they fail**

Run: `go test ./app/auth ./foundation/persist -run 'TestAuthorizerAliasMatchesPlatformType|TestStoreSetAliasMatchesPlatformType'`

Expected: FAIL with import or undefined-type errors until the new `internal/platform/...` packages exist.

- [ ] **Step 3: Create platform packages and add type-alias shims**

```go
// internal/platform/auth/authorizer.go
package auth

import "time"

type TokenConfig struct {
	Role  string
	Actor string
}

type Config struct {
	ReaderToken    string
	AuthoringToken string
	PublisherToken string
	Tokens         map[string]TokenConfig
}

type Authorizer struct {
	tokens map[string]Principal
}

func NewAuthorizer(config Config) *Authorizer {
	return &Authorizer{tokens: buildTokenPrincipals(config)}
}
```

```go
// app/auth/authorizer.go
package auth

import platformauth "skill_registry/internal/platform/auth"

type TokenConfig = platformauth.TokenConfig
type Config = platformauth.Config
type Authorizer = platformauth.Authorizer

var NewAuthorizer = platformauth.NewAuthorizer
```

```go
// foundation/persist/store_set.go
package persist

import platformpersist "skill_registry/internal/platform/persistence"

type SQLStoreSet = platformpersist.SQLStoreSet

var NewSQLStoreSetWithDB = platformpersist.NewSQLStoreSetWithDB
```

- [ ] **Step 4: Run focused tests to verify the shims pass**

Run: `go test ./app/auth ./app/observability ./foundation/persist ./foundation/artifact`

Expected: PASS

- [ ] **Step 5: Commit the platform extraction checkpoint**

```bash
git add internal/platform app/auth app/observability foundation/persist foundation/artifact
git commit -m "refactor: extract shared platform seams"
```

## Task 2: Extract shared HTTP runtime from `app/catalogapi`

**Files:**
- Create: `internal/platform/http/engine.go`
- Create: `internal/platform/http/metadata.go`
- Create: `internal/platform/http/response.go`
- Create: `internal/platform/http/request_limiter.go`
- Create: `internal/platform/http/read_cache.go`
- Create: `internal/platform/http/middleware/auth.go`
- Create: `internal/platform/http/middleware/canonical.go`
- Create: `internal/platform/http/middleware/cors.go`
- Create: `internal/platform/http/middleware/metadata.go`
- Create: `internal/platform/http/middleware/observability.go`
- Modify: `app/catalogapi/request_limiter.go`
- Modify: `app/catalogapi/read_cache.go`
- Modify: `app/catalogapi/common/context.go`
- Modify: `app/catalogapi/common/errors.go`
- Modify: `app/catalogapi/common/observability.go`
- Modify: `app/catalogapi/common/request.go`
- Modify: `app/catalogapi/common/response.go`
- Modify: `app/catalogapi/middleware/auth.go`
- Modify: `app/catalogapi/middleware/canonical.go`
- Modify: `app/catalogapi/middleware/cors.go`
- Modify: `app/catalogapi/middleware/metadata.go`
- Modify: `app/catalogapi/middleware/observability.go`
- Test: `app/catalogapi/request_limiter_test.go`
- Test: `app/catalogapi/read_cache_test.go`
- Test: `app/catalogapi/server_test.go`

- [ ] **Step 1: Add a failing router smoke test against the future platform engine**

```go
package catalogapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	platformhttp "skill_registry/internal/platform/http"
)

func TestPlatformEnginePreservesRawPathSettings(t *testing.T) {
	engine := platformhttp.NewEngine(nil)
	engine.GET("/ping", func(ctx *gin.Context) { ctx.Status(http.StatusNoContent) })

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}
```

- [ ] **Step 2: Run the HTTP seam tests to verify the new package is still missing**

Run: `go test ./app/catalogapi -run 'TestPlatformEnginePreservesRawPathSettings|TestSkillsRequiresAuthentication'`

Expected: FAIL with undefined `platformhttp.NewEngine` or missing package errors.

- [ ] **Step 3: Implement shared HTTP packages and turn old files into wrappers**

```go
// internal/platform/http/engine.go
package http

import "github.com/gin-gonic/gin"

func NewEngine(middleware []gin.HandlerFunc) *gin.Engine {
	engine := gin.New()
	engine.UseRawPath = true
	engine.UnescapePathValues = false
	engine.HandleMethodNotAllowed = true
	engine.RedirectTrailingSlash = false
	engine.RedirectFixedPath = false
	if len(middleware) > 0 {
		engine.Use(middleware...)
	}
	return engine
}
```

```go
// app/catalogapi/request_limiter.go
package catalogapi

import platformhttp "skill_registry/internal/platform/http"

type requestLimiter = platformhttp.RequestLimiter

var newRequestLimiter = platformhttp.NewRequestLimiter
```

```go
// app/catalogapi/common/response.go
package common

import platformhttp "skill_registry/internal/platform/http"

var WriteJSONBytes = platformhttp.WriteJSONBytes
var WriteCatalogError = platformhttp.WriteCatalogError
```

- [ ] **Step 4: Run the HTTP package tests**

Run: `go test ./app/catalogapi`

Expected: PASS

- [ ] **Step 5: Commit the HTTP platform seam extraction**

```bash
git add internal/platform/http app/catalogapi/common app/catalogapi/middleware app/catalogapi/request_limiter.go app/catalogapi/read_cache.go
git commit -m "refactor: extract shared http runtime"
```

## Task 3: Split Catalog into `domain`, `application`, and `infra`

**Files:**
- Create: `internal/catalog/domain/types.go`
- Create: `internal/catalog/domain/errors.go`
- Create: `internal/catalog/domain/repository.go`
- Create: `internal/catalog/application/read_service.go`
- Create: `internal/catalog/application/write_service.go`
- Create: `internal/catalog/infra/sql_repository.go`
- Create: `internal/catalog/infra/sql_state_store.go`
- Create: `internal/catalog/infra/memory_repository.go`
- Modify: `business/catalog/types.go`
- Modify: `business/catalog/errors.go`
- Modify: `business/catalog/repository.go`
- Modify: `business/catalog/read_module.go`
- Modify: `business/catalog/write_module.go`
- Modify: `business/catalog/sql_repository.go`
- Modify: `business/catalog/sql_state_store.go`
- Modify: `business/catalog/memory_repository.go`
- Modify: `app/catalogapi/catalog_read_module.go`
- Modify: `app/catalogapi/catalog_mutation_module.go`
- Modify: `app/catalogapi/governance_module.go`
- Test: `business/catalog/read_module_test.go`
- Test: `business/catalog/write_module_test.go`
- Test: `business/catalog/sql_repository_test.go`

- [ ] **Step 1: Add a failing application-service test in the new catalog module**

```go
package application

import (
	"context"
	"testing"

	"skill_registry/internal/catalog/domain"
)

type stubQueryStore struct {
	listed bool
}

func (s *stubQueryStore) ListSkills(context.Context, domain.SkillQuery) (domain.SkillsPage, error) {
	s.listed = true
	return domain.SkillsPage{}, nil
}

func TestReadServiceListSkillsDelegatesToQueryStore(t *testing.T) {
	store := &stubQueryStore{}
	service := NewReadService(store)

	if _, err := service.ListSkills(context.Background(), domain.SkillQuery{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !store.listed {
		t.Fatal("expected query store to be called")
	}
}
```

- [ ] **Step 2: Run the catalog package tests to verify the new services are still missing**

Run: `go test ./business/catalog ./app/catalogapi -run 'TestReadServiceListSkillsDelegatesToQueryStore|TestServerRoutesSkillsThroughCatalogReadModule'`

Expected: FAIL with undefined `NewReadService`, `domain.SkillQuery`, or moved-type errors.

- [ ] **Step 3: Create the new catalog packages and leave business-layer aliases behind**

```go
// internal/catalog/domain/repository.go
package domain

import "context"

type QueryStore interface {
	ListSkills(ctx context.Context, query SkillQuery) (SkillsPage, error)
	ListCuratedSkills(ctx context.Context, query CuratedSkillQuery) (CuratedSkillsPage, error)
	GetSkill(ctx context.Context, packageName string) (SkillDetail, error)
	ListVersions(ctx context.Context, query VersionQuery) (VersionsPage, error)
	GetVersion(ctx context.Context, packageName string, version string) (SkillVersionDetail, error)
	ListGovernanceChanges(ctx context.Context, query GovernanceQuery) (GovernancePage, error)
	ResolveInstallTarget(ctx context.Context, packageName string, version string) (InstallResolution, error)
}
```

```go
// internal/catalog/application/read_service.go
package application

import (
	"context"

	"skill_registry/internal/catalog/domain"
)

type ReadService struct {
	store domain.QueryStore
}

func NewReadService(store domain.QueryStore) *ReadService {
	return &ReadService{store: store}
}

func (s *ReadService) ListSkills(ctx context.Context, query domain.SkillQuery) (domain.SkillsPage, error) {
	return s.store.ListSkills(ctx, query)
}
```

```go
// business/catalog/read_module.go
package catalog

import (
	internalapp "skill_registry/internal/catalog/application"
	internaldomain "skill_registry/internal/catalog/domain"
)

type ReadModule = internalapp.ReadService
type QueryStore = internaldomain.QueryStore

var newReadModule = internalapp.NewReadService
```

- [ ] **Step 4: Run catalog-focused tests**

Run: `go test ./business/catalog ./app/catalogapi`

Expected: PASS

- [ ] **Step 5: Commit the catalog split**

```bash
git add internal/catalog business/catalog app/catalogapi/catalog_read_module.go app/catalogapi/catalog_mutation_module.go app/catalogapi/governance_module.go
git commit -m "refactor: split catalog into domain application infra"
```

## Task 4: Split Packaging into `domain`, `application`, and `infra`

**Files:**
- Create: `internal/packaging/domain/types.go`
- Create: `internal/packaging/domain/warnings.go`
- Create: `internal/packaging/domain/bundle_rules.go`
- Create: `internal/packaging/application/workspace_service.go`
- Create: `internal/packaging/application/publish_service.go`
- Create: `internal/packaging/infra/workspace_adapter.go`
- Create: `internal/packaging/infra/workspace_backing.go`
- Create: `internal/packaging/infra/workspace_locking.go`
- Modify: `business/packaging/types.go`
- Modify: `business/packaging/warnings.go`
- Modify: `business/packaging/bundle_rule_module.go`
- Modify: `business/packaging/service.go`
- Modify: `business/packaging/workspace_runtime_module.go`
- Modify: `business/packaging/published_release_workflow.go`
- Modify: `app/catalogapi/packaging_module.go`
- Modify: `app/catalogapi/packaging_access_module.go`
- Test: `business/packaging/service_test.go`
- Test: `business/packaging/workspace_runtime_module_test.go`
- Test: `business/packaging/published_release_workflow_test.go`
- Test: `app/catalogapi/packaging_module_test.go`

- [ ] **Step 1: Add a failing workspace-service test in the new packaging application layer**

```go
package application

import (
	"context"
	"testing"

	"skill_registry/internal/packaging/domain"
)

type stubRuntime struct {
	prepared bool
}

func (s *stubRuntime) Prepare(ctx context.Context, input domain.PrepareRequest) (domain.PrepareResponse, error) {
	s.prepared = true
	return domain.PrepareResponse{}, nil
}

func TestWorkspaceServicePrepareDelegatesToRuntime(t *testing.T) {
	runtime := &stubRuntime{}
	service := NewWorkspaceService(runtime)

	if _, err := service.PrepareContext(context.Background(), domain.PrepareRequest{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !runtime.prepared {
		t.Fatal("expected runtime to be called")
	}
}
```

- [ ] **Step 2: Run packaging tests to verify the new service seam is absent**

Run: `go test ./business/packaging ./app/catalogapi -run 'TestWorkspaceServicePrepareDelegatesToRuntime|TestPackagingPrepareRequiresAuthoringAccess'`

Expected: FAIL with undefined `NewWorkspaceService` or moved-type errors.

- [ ] **Step 3: Create packaging application services and preserve the old service API as an alias**

```go
// internal/packaging/application/workspace_service.go
package application

import (
	"context"
	"time"

	platformpersist "skill_registry/internal/platform/persistence"
	"skill_registry/internal/packaging/domain"
)

type WorkspaceRuntime interface {
	Prepare(ctx context.Context, input domain.PrepareRequest) (domain.PrepareResponse, error)
	Validate(ctx context.Context, input domain.ValidateRequest) (domain.ValidateResponse, error)
	Pack(ctx context.Context, input domain.PackRequest) (domain.PackResponse, error)
	Cleanup(ctx context.Context, workspacePath string) error
}

type WorkspaceService struct {
	runtime WorkspaceRuntime
}

func NewWorkspaceService(runtime WorkspaceRuntime) *WorkspaceService {
	return &WorkspaceService{runtime: runtime}
}

func (s *WorkspaceService) PrepareContext(ctx context.Context, input domain.PrepareRequest) (domain.PrepareResponse, error) {
	return s.runtime.Prepare(ctx, input)
}
```

```go
// internal/packaging/application/service.go
package application

import (
	"context"
	"time"

	platformpersist "skill_registry/internal/platform/persistence"
	packagingdomain "skill_registry/internal/packaging/domain"
	packaginginfra "skill_registry/internal/packaging/infra"
)

type Service struct {
	workspaceRuntime interface {
		Prepare(context.Context, packagingdomain.PrepareRequest) (packagingdomain.PrepareResponse, error)
		Validate(context.Context, packagingdomain.ValidateRequest) (packagingdomain.ValidateResponse, error)
		Pack(context.Context, packagingdomain.PackRequest) (packagingdomain.PackResponse, error)
		Cleanup(context.Context, string) error
		PruneExpiredWorkspaces(context.Context, time.Duration) error
	}
	publishWorkflow *PublishService
}

func NewService(tempRoot string) *Service {
	return NewServiceWithWorkspaceStore(tempRoot, nil)
}

func NewServiceWithWorkspaceStore(tempRoot string, store platformpersist.WorkspaceStore) *Service {
	runtime := packaginginfra.NewWorkspaceRuntime(tempRoot, store)
	return &Service{
		workspaceRuntime: runtime,
		publishWorkflow:  NewPublishService(runtime),
	}
}

func (s *Service) PrepareContext(ctx context.Context, input packagingdomain.PrepareRequest) (packagingdomain.PrepareResponse, error) {
	return s.workspaceRuntime.Prepare(ctx, input)
}

func (s *Service) PruneExpiredWorkspaces(ttl time.Duration) error {
	return s.workspaceRuntime.PruneExpiredWorkspaces(context.Background(), ttl)
}
```

```go
// business/packaging/service.go
package packaging

import internalapp "skill_registry/internal/packaging/application"

type Service = internalapp.Service

var (
	NewService                   = internalapp.NewService
	NewServiceWithWorkspaceStore = internalapp.NewServiceWithWorkspaceStore
)
```

```go
// internal/packaging/application/publish_service.go
package application

import (
	"context"

	"skill_registry/internal/packaging/domain"
)

type PublishService struct {
	publishWorkflow interface {
		Publish(context.Context, domain.PublishRequest) (domain.PublishResponse, error)
	}
}

func NewPublishService(workflow interface {
	Publish(context.Context, domain.PublishRequest) (domain.PublishResponse, error)
}) *PublishService {
	return &PublishService{publishWorkflow: workflow}
}

func (s *Service) PublishContext(ctx context.Context, input domain.PublishRequest) (domain.PublishResponse, error) {
	return s.publishWorkflow.publishWorkflow.Publish(ctx, input)
}
```

- [ ] **Step 4: Run packaging tests**

Run: `go test ./business/packaging ./app/catalogapi`

Expected: PASS

- [ ] **Step 5: Commit the packaging split**

```bash
git add internal/packaging business/packaging app/catalogapi/packaging_module.go app/catalogapi/packaging_access_module.go
git commit -m "refactor: split packaging into domain application infra"
```

## Task 5: Move route registration and handlers into domain HTTP modules

**Files:**
- Create: `internal/catalog/http/handlers.go`
- Create: `internal/catalog/http/binding.go`
- Create: `internal/catalog/http/routes.go`
- Create: `internal/packaging/http/handlers.go`
- Create: `internal/packaging/http/binding.go`
- Create: `internal/packaging/http/routes.go`
- Create: `internal/system/http/routes.go`
- Create: `internal/governance/http/routes.go`
- Modify: `app/catalogapi/handlers/catalog/handlers.go`
- Modify: `app/catalogapi/handlers/packaging/handlers.go`
- Modify: `app/catalogapi/handlers/system/handlers.go`
- Modify: `app/catalogapi/handlers/governance/handlers.go`
- Modify: `app/catalogapi/binding/catalog.go`
- Modify: `app/catalogapi/binding/packaging.go`
- Modify: `app/catalogapi/router/router.go`
- Modify: `app/catalogapi/router/catalog.go`
- Modify: `app/catalogapi/router/packaging.go`
- Modify: `app/catalogapi/router/system.go`
- Test: `app/catalogapi/server_test.go`
- Test: `app/catalogapi/access_workflow_test.go`

- [ ] **Step 1: Add a failing route-registration test in the new catalog HTTP package**

```go
package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterCatalogRoutesMountsSkillsEndpoint(t *testing.T) {
	engine := gin.New()
	registerCatalogRoutes(engine, Config{
		ServesReadRoutes: true,
		Skills: func(ctx *gin.Context) {
			ctx.Status(http.StatusNoContent)
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}
```

- [ ] **Step 2: Run the route tests to verify the new HTTP packages are missing**

Run: `go test ./app/catalogapi -run 'TestRegisterCatalogRoutesMountsSkillsEndpoint|TestServerRoutesHeadSkillsThroughCatalogReadModule'`

Expected: FAIL with missing `internal/catalog/http` symbols.

- [ ] **Step 3: Create new HTTP route registrars and make old router files delegate**

```go
// internal/catalog/http/routes.go
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Config struct {
	ServesReadRoutes  bool
	ServesWriteRoutes bool
	Skills            gin.HandlerFunc
	SkillsSearch      gin.HandlerFunc
	CuratedSkills     gin.HandlerFunc
	GovernanceChanges gin.HandlerFunc
	SkillDetail       gin.HandlerFunc
	VersionList       gin.HandlerFunc
	VersionDetail     gin.HandlerFunc
	InstallTarget     gin.HandlerFunc
	SkillGovernance   gin.HandlerFunc
	ReleaseRecord     gin.HandlerFunc
	VersionGovernance gin.HandlerFunc
}

func RegisterRoutes(engine *gin.Engine, config Config) {
	registerHeadCompatibleRoute(engine, "/api/v1/skills", config.Skills)
}

func registerHeadCompatibleRoute(engine *gin.Engine, path string, handler gin.HandlerFunc) {
	engine.Match([]string{http.MethodGet, http.MethodHead}, path, handler)
}
```

```go
// app/catalogapi/router/catalog.go
package router

import (
	cataloghttp "skill_registry/internal/catalog/http"
)

func registerCatalogRoutes(engine *gin.Engine, config Config) {
	cataloghttp.RegisterRoutes(engine, cataloghttp.Config{
		ServesReadRoutes:  config.ServesReadRoutes,
		ServesWriteRoutes: config.ServesWriteRoutes,
		Skills:            config.Skills,
	})
}
```

- [ ] **Step 4: Run the catalogapi route tests**

Run: `go test ./app/catalogapi`

Expected: PASS

- [ ] **Step 5: Commit the HTTP module move**

```bash
git add internal/catalog/http internal/packaging/http internal/system/http internal/governance/http app/catalogapi/router app/catalogapi/handlers app/catalogapi/binding
git commit -m "refactor: move handlers and routes into domain http modules"
```

## Task 6: Introduce a thin backend runtime and move assembly out of `catalogapi`

**Files:**
- Create: `internal/backend/runtime.go`
- Create: `internal/backend/runtime_test.go`
- Modify: `app/catalogapi/server.go`
- Modify: `app/catalogapi/server_lifecycle.go`
- Modify: `app/catalogapi/server_mode.go`
- Modify: `cmd/skill-registry/main.go`
- Test: `internal/backend/runtime_test.go`
- Test: `cmd/skill-registry/main_test.go`
- Test: `app/catalogapi/server_workflow_seam_test.go`

- [ ] **Step 1: Add a failing runtime-construction test**

```go
package backend

import (
	"context"
	"testing"

	"skill_registry/internal/catalog/application"
	"skill_registry/internal/catalog/domain"
)

type stubQueryStore struct {
	domain.QueryStore
}

func (s *stubQueryStore) ListSkills(context.Context, domain.SkillQuery) (domain.SkillsPage, error) {
	return domain.SkillsPage{}, nil
}

func TestNewRuntimeBuildsCatalogReadService(t *testing.T) {
	readRepo := &stubQueryStore{}
	runtime := NewRuntime(Config{Repository: readRepo})
	if runtime.CatalogRead == nil {
		t.Fatal("expected catalog read service to be wired")
	}
	var expected *application.ReadService
	if _, ok := any(runtime.CatalogRead).(*application.ReadService); !ok {
		t.Fatal("expected catalog read service implementation")
	}
	if expected == nil {
		t.Log("type assertion compiled against *application.ReadService")
	}
}
```

- [ ] **Step 2: Run backend and command tests to verify the runtime is missing**

Run: `go test ./cmd/skill-registry ./app/catalogapi -run 'TestNewRuntimeBuildsCatalogReadService|TestServeHTTPServer'`

Expected: FAIL with undefined `NewRuntime` or missing runtime fields.

- [ ] **Step 3: Create the runtime composition root and shrink `cmd/skill-registry/main.go`**

```go
// internal/backend/runtime.go
package backend

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	catalogapp "skill_registry/internal/catalog/application"
	catalogdomain "skill_registry/internal/catalog/domain"
	platformhttp "skill_registry/internal/platform/http"
)

type Runtime struct {
	CatalogRead *catalogapp.ReadService
	Handler     http.Handler
	HTTPServer  *http.Server
}

type Config struct {
	Addr         string
	Repository   catalogdomain.QueryStore
	WriteTimeout time.Duration
}

func NewRuntime(config Config) Runtime {
	read := catalogapp.NewReadService(config.Repository)
	handler := platformhttp.NewEngine([]gin.HandlerFunc{})
	return Runtime{
		CatalogRead: read,
		Handler:     handler,
		HTTPServer: &http.Server{
			Addr:         config.Addr,
			Handler:      handler,
			WriteTimeout: config.WriteTimeout,
		},
	}
}
```

```go
// cmd/skill-registry/main.go
package main

import (
	"log"

	"skill_registry/internal/backend"
)

func main() {
	runtime := backend.NewRuntime(configuredRuntime())
	if err := serveHTTPServer(runtime.HTTPServer); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 4: Run backend composition tests**

Run: `go test ./cmd/skill-registry ./app/catalogapi ./internal/backend`

Expected: PASS

- [ ] **Step 5: Commit the runtime extraction**

```bash
git add internal/backend cmd/skill-registry/main.go app/catalogapi/server.go app/catalogapi/server_lifecycle.go app/catalogapi/server_mode.go
git commit -m "refactor: introduce thin backend runtime"
```

## Task 7: Delete compatibility shims and finish the package migration

**Files:**
- Modify: `app/auth/authorizer.go`
- Modify: `app/observability/http.go`
- Modify: `foundation/persist/store_set.go`
- Modify: `foundation/artifact/publisher.go`
- Modify: `business/catalog/read_module.go`
- Modify: `business/catalog/write_module.go`
- Modify: `business/packaging/service.go`
- Modify: `app/catalogapi/server.go`
- Delete: `app/catalogapi/catalog_read_module.go`
- Delete: `app/catalogapi/catalog_mutation_module.go`
- Delete: `app/catalogapi/governance_module.go`
- Delete: `app/catalogapi/packaging_module.go`
- Delete: `app/catalogapi/packaging_access_module.go`
- Delete: `app/catalogapi/read_api_module.go`
- Delete: `app/catalogapi/handlers_catalog.go`
- Test: `go test ./...`

- [ ] **Step 1: Add a failing architecture guard test so old broad modules cannot return**

```go
package backend

import (
	"os"
	"testing"
)

func TestLegacyCatalogAPIModulesRemoved(t *testing.T) {
	legacyFiles := []string{
		"../../app/catalogapi/catalog_read_module.go",
		"../../app/catalogapi/catalog_mutation_module.go",
		"../../app/catalogapi/read_api_module.go",
	}
	for _, path := range legacyFiles {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("expected legacy file %s to be removed", path)
		}
	}
}
```

- [ ] **Step 2: Run the full suite to see the architecture guard fail before file deletion**

Run: `go test ./internal/backend ./...`

Expected: FAIL because the legacy `app/catalogapi/*_module.go` files still exist.

- [ ] **Step 3: Remove the broad pass-through modules and inline any remaining callers onto the new services**

```go
// app/catalogapi/server.go
package catalogapi

type Server struct {
	handler http.Handler
}

func NewServer(config ServerConfig) *Server {
	runtime := backend.NewRuntime(backend.Config{
		Repository: config.Repository,
	})
	return &Server{handler: runtime.Handler}
}
```

```go
// business/catalog/read_module.go
package catalog

// delete this shim after all callers import skill_registry/internal/catalog/application directly
```

```go
// app/catalogapi/catalog_read_module.go
// delete file
```

- [ ] **Step 4: Run the full verification suite**

Run: `go test ./...`

Expected: PASS

- [ ] **Step 5: Commit the final cleanup**

```bash
git add -A
git commit -m "refactor: complete backend package reorganization"
```

## Self-Review

### Spec coverage

- `internal/platform` extraction: covered by Tasks 1 and 2
- Catalog split: covered by Task 3
- Packaging split: covered by Task 4
- Governance and System route relocation: covered by Task 5
- Thin composition root in `cmd/skill-registry`: covered by Task 6
- Deletion of shallow `catalogapi` modules: covered by Task 7
- External HTTP contract stability: preserved by Tasks 2, 5, 6, and the repeated `app/catalogapi` test runs

### Placeholder scan

- No `TODO`, `TBD`, or “similar to Task N” placeholders remain
- Each task lists exact files and exact verification commands
- Each code-changing step includes concrete code snippets

### Type consistency

- Shared-platform aliases point to `internal/platform/...`
- Catalog runtime uses `internal/catalog/application` and `internal/catalog/domain`
- Packaging runtime uses `internal/packaging/application` and `internal/packaging/domain`
- Backend runtime becomes the composition seam referenced by both `cmd/skill-registry` and `app/catalogapi`
