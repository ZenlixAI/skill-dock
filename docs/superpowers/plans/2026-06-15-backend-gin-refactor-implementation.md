# Backend Gin Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the backend HTTP adapter to `gin`, split `app/catalogapi` by HTTP responsibility and feature, and simplify `Makefile` so only meaningful dev, build, test, verify, and deployment targets remain.

**Architecture:** Keep the existing `Server` dependency assembly and business workflows, but replace `http.ServeMux` plus custom wrappers with a gin-based router, middleware chain, and binding layer. Migrate one endpoint group at a time, preserving API paths, auth behavior, error payloads, and test coverage while removing obsolete adapter glue after each replacement is verified.

**Tech Stack:** Go 1.26, `github.com/gin-gonic/gin`, `net/http`, existing catalog/packaging/auth modules, GNU Make

---

### Task 1: Add Gin And Establish The Router Shell

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Modify: `app/catalogapi/server.go`
- Create: `app/catalogapi/router/router.go`
- Create: `app/catalogapi/router/catalog.go`
- Create: `app/catalogapi/router/session.go`
- Create: `app/catalogapi/router/system.go`
- Create: `app/catalogapi/router/packaging.go`
- Test: `app/catalogapi/server_test.go`

- [ ] **Step 1: Add a failing routing test that still exercises the public `Server.Handler()` entrypoint**

```go
func TestServerHandlerExposesGinBackedRoutes(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}
```

- [ ] **Step 2: Run the focused route test to verify the baseline still passes before migration**

Run: `env GOCACHE=/private/tmp/go-cache go test ./app/catalogapi -run TestServerHandlerExposesGinBackedRoutes -count=1`
Expected: `PASS`

- [ ] **Step 3: Add `gin` to the module and create a router package that returns a configured `*gin.Engine`**

```go
func Build(server *catalogapi.Server) http.Handler {
	engine := gin.New()
	registerSystemRoutes(engine, server)
	registerSessionRoutes(engine, server)
	registerCatalogRoutes(engine, server)
	registerPackagingRoutes(engine, server)
	return engine
}
```

- [ ] **Step 4: Update `Server.Handler()` to delegate route construction to the new router package while preserving `http.Handler` as the exported type**

```go
func (s *Server) Handler() http.Handler {
	return router.Build(s)
}
```

- [ ] **Step 5: Re-run the focused route tests**

Run: `env GOCACHE=/private/tmp/go-cache go test ./app/catalogapi -run 'TestServerHandlerExposesGinBackedRoutes|TestSkillsRequiresAuthentication' -count=1`
Expected: `PASS`

- [ ] **Step 6: Commit the shell migration**

```bash
git add go.mod go.sum app/catalogapi/server.go app/catalogapi/router app/catalogapi/server_test.go
git commit -m "refactor: add gin router shell for catalog api"
```

### Task 2: Move Shared HTTP Concerns Into Gin Middleware

**Files:**
- Create: `app/catalogapi/middleware/cors.go`
- Create: `app/catalogapi/middleware/metadata.go`
- Create: `app/catalogapi/middleware/observability.go`
- Create: `app/catalogapi/middleware/auth.go`
- Modify: `app/catalogapi/server.go`
- Modify: `app/catalogapi/access_workflow.go`
- Test: `app/catalogapi/access_workflow_test.go`
- Test: `app/catalogapi/server_test.go`

- [ ] **Step 1: Add failing tests for CORS preflight, trace propagation, and unauthorized protected routes through the new gin stack**

```go
func TestProtectedGinRouteRejectsAnonymousRequest(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}
```

- [ ] **Step 2: Run the focused middleware tests**

Run: `env GOCACHE=/private/tmp/go-cache go test ./app/catalogapi -run 'TestSkillsRequiresAuthentication|TestAccessWorkflowDeniesPackageAccessAndRecordsAudit' -count=1`
Expected: `PASS`

- [ ] **Step 3: Port request metadata, CORS, observability, and authentication wrappers into gin middleware**

```go
func RequestMetadata(server *catalogapi.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := "req_" + strconv.FormatUint(server.NextRequestID(), 10)
		traceID := resolveTraceID(c.Request.Header.Get("X-Trace-Id"), requestID)
		c.Writer.Header().Set("X-Request-Id", requestID)
		c.Writer.Header().Set("X-Trace-Id", traceID)
		ctx := common.ContextWithRequestMetadata(c.Request.Context(), requestID, traceID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
```

- [ ] **Step 4: Add gin middleware helpers for principal injection and action guards while keeping `accessWorkflow` as the authorization core**

```go
func RequireAction(server *catalogapi.Server, action auth.Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := server.AuthenticateRequest(c.Request)
		if !ok {
			response.WriteCatalogError(c, catalog.NewError(401, "unauthorized", "Authentication is required.", nil))
			c.Abort()
			return
		}
		c.Request = c.Request.WithContext(auth.ContextWithPrincipal(c.Request.Context(), principal))
		if !server.AuthorizeAction(c.Writer, c.Request, action, "", "") {
			c.Abort()
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 5: Re-run middleware and auth tests**

Run: `env GOCACHE=/private/tmp/go-cache go test ./app/catalogapi -run 'TestSkillsRequiresAuthentication|TestBearerTokenAuthUsesConfiguredRoleMapping|TestAccessWorkflowDeniesPackageAccessAndRecordsAudit' -count=1`
Expected: `PASS`

- [ ] **Step 6: Commit the middleware migration**

```bash
git add app/catalogapi/middleware app/catalogapi/server.go app/catalogapi/access_workflow.go app/catalogapi/access_workflow_test.go app/catalogapi/server_test.go
git commit -m "refactor: move catalog api wrappers into gin middleware"
```

### Task 3: Migrate System And Session Endpoints

**Files:**
- Create: `app/catalogapi/handlers/system/handlers.go`
- Create: `app/catalogapi/handlers/session/handlers.go`
- Create: `app/catalogapi/response/json.go`
- Create: `app/catalogapi/response/errors.go`
- Modify: `app/catalogapi/router/system.go`
- Modify: `app/catalogapi/router/session.go`
- Modify: `app/catalogapi/system/module.go`
- Modify: `app/catalogapi/session.go`
- Test: `app/catalogapi/server_test.go`
- Test: `app/catalogapi/session_module_test.go`
- Test: `app/catalogapi/system/module_test.go`

- [ ] **Step 1: Add failing tests covering session login/logout/current and health/ready/metrics through the gin handlers**

```go
func TestSessionEndpointReturnsCurrentPrincipal(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository:     catalog.NewMemoryRepository(catalog.SampleData()),
		ReaderToken:    "reader-secret",
		PublisherToken: "publisher-secret",
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	request.Header.Set("Authorization", "Bearer publisher-secret")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}
```

- [ ] **Step 2: Run focused session and system tests**

Run: `env GOCACHE=/private/tmp/go-cache go test ./app/catalogapi -run 'TestSessionEndpointReturnsCurrentPrincipal|TestSessionEndpointRequiresAuthentication|TestReadiness' -count=1`
Expected: `PASS`

- [ ] **Step 3: Introduce response helpers for gin-backed JSON and catalog errors**

```go
func WriteJSON(c *gin.Context, status int, payload any) {
	c.JSON(status, payload)
}
```

- [ ] **Step 4: Move system and session handlers into feature directories and register them via gin routes**

```go
group := engine.Group("/api/v1/session")
group.GET("", sessionHandlers.Current)
group.POST("/login", sessionHandlers.Login)
group.POST("/logout", sessionHandlers.Logout)
```

- [ ] **Step 5: Re-run focused tests**

Run: `env GOCACHE=/private/tmp/go-cache go test ./app/catalogapi -run 'TestSessionEndpointReturnsCurrentPrincipal|TestSessionEndpointRequiresAuthentication|TestSessionLoginRejectsInvalidJSON|TestReadiness' -count=1`
Expected: `PASS`

- [ ] **Step 6: Commit the session/system migration**

```bash
git add app/catalogapi/handlers/system app/catalogapi/handlers/session app/catalogapi/response app/catalogapi/router/system.go app/catalogapi/router/session.go app/catalogapi/session.go app/catalogapi/system/module.go app/catalogapi/server_test.go app/catalogapi/session_module_test.go app/catalogapi/system/module_test.go
git commit -m "refactor: move system and session endpoints to gin handlers"
```

### Task 4: Migrate Catalog Read Endpoints With Gin Binding

**Files:**
- Create: `app/catalogapi/handlers/catalog/handlers.go`
- Create: `app/catalogapi/binding/catalog.go`
- Modify: `app/catalogapi/router/catalog.go`
- Modify: `app/catalogapi/common/params.go`
- Modify: `app/catalogapi/handlers_catalog.go`
- Modify: `app/catalogapi/server_test.go`
- Modify: `app/catalogapi/read_api_module.go`
- Test: `app/catalogapi/server_test.go`
- Test: `app/catalogapi/read_cache_test.go`

- [ ] **Step 1: Add failing tests that prove query and path parameters survive the gin migration**

```go
func TestReadAPIModuleListsSkillsFromParsedQuery(t *testing.T) {
	// existing expectations remain the contract
}
```

- [ ] **Step 2: Run the focused catalog read tests**

Run: `env GOCACHE=/private/tmp/go-cache go test ./app/catalogapi -run 'TestServerRoutesSkillsThroughCatalogReadModule|TestReadAPIModuleListsSkillsFromParsedQuery|TestSkillsRequiresAuthentication' -count=1`
Expected: `PASS`

- [ ] **Step 3: Introduce gin query/path binding structs for transport values and keep helper validation for cursor, semver, and enums**

```go
type SkillsQuery struct {
	Query          string `form:"q"`
	PackageName    string `form:"package_name"`
	SourceType     string `form:"source_type"`
	IncludeRetired bool   `form:"include_retired"`
	Sort           string `form:"sort"`
	PageSize       int    `form:"pageSize"`
	Cursor         string `form:"cursor"`
}
```

- [ ] **Step 4: Move skills, curated, audit, governance list, detail, versions, and install-target handlers into the catalog handler package**

```go
catalogRoutes.GET("/skills", middleware.RequireAction(server, auth.ActionReadCatalog), handlers.ListSkills)
catalogRoutes.GET("/skills/:package", middleware.RequireAction(server, auth.ActionReadCatalog), handlers.GetSkill)
catalogRoutes.GET("/skills/:package/versions/:version", middleware.RequireAction(server, auth.ActionReadCatalog), handlers.GetVersion)
```

- [ ] **Step 5: Re-run focused catalog read tests**

Run: `env GOCACHE=/private/tmp/go-cache go test ./app/catalogapi -run 'TestServerRoutesSkillsThroughCatalogReadModule|TestReadAPIModuleListsSkillsFromParsedQuery|TestBearerTokenAuthUsesConfiguredRoleMapping' -count=1`
Expected: `PASS`

- [ ] **Step 6: Commit the catalog read migration**

```bash
git add app/catalogapi/handlers/catalog app/catalogapi/binding/catalog.go app/catalogapi/router/catalog.go app/catalogapi/common/params.go app/catalogapi/handlers_catalog.go app/catalogapi/read_api_module.go app/catalogapi/server_test.go app/catalogapi/read_cache_test.go
git commit -m "refactor: migrate catalog read endpoints to gin"
```

### Task 5: Migrate Governance And Packaging Write Endpoints

**Files:**
- Create: `app/catalogapi/handlers/governance/handlers.go`
- Create: `app/catalogapi/handlers/packaging/handlers.go`
- Create: `app/catalogapi/binding/packaging.go`
- Modify: `app/catalogapi/router/catalog.go`
- Modify: `app/catalogapi/router/packaging.go`
- Modify: `app/catalogapi/handlers_mutation.go`
- Modify: `app/catalogapi/handlers_packaging.go`
- Modify: `app/catalogapi/packaging_module.go`
- Modify: `app/catalogapi/packaging_access_module.go`
- Test: `app/catalogapi/packaging_module_test.go`
- Test: `app/catalogapi/packaging_access_module_test.go`
- Test: `app/catalogapi/server_test.go`

- [ ] **Step 1: Add failing tests for JSON binding and package-scoped authorization on write endpoints**

```go
func TestPackagingPrepareRejectsInvalidJSON(t *testing.T) {
	server := newPackagingEnabledTestServer(t)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", strings.NewReader("{"))
	request.Header.Set("Authorization", "Bearer authoring-secret")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, recorder.Code)
	}
}
```

- [ ] **Step 2: Run focused write-path tests**

Run: `env GOCACHE=/private/tmp/go-cache go test ./app/catalogapi -run 'TestPackagingModuleReplaysCachedPrepareResponse|TestPackagingPrepareRejectsInvalidJSON|TestBearerTokenAuthRejectsXRoleOverrideWhenMappingConfigured' -count=1`
Expected: `PASS`

- [ ] **Step 3: Move governance and packaging handlers into gin packages and use gin JSON binding where transport structs map cleanly**

```go
func (h *Handlers) Prepare(c *gin.Context) {
	var payload packaging.PrepareRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		response.WriteCatalogError(c, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", nil))
		return
	}
	// preserve package authorization and module execution
}
```

- [ ] **Step 4: Keep special cases explicit: write limiter usage, bundle streaming, idempotency, and audit recording**

```go
func (h *Handlers) DownloadBundle(c *gin.Context) {
	bundlePath, err := h.packagingAccess.downloadBundle(...)
	if err != nil {
		response.WriteCatalogError(c, err)
		return
	}
	c.FileAttachment(bundlePath, filepath.Base(bundlePath))
}
```

- [ ] **Step 5: Re-run focused packaging and mutation tests**

Run: `env GOCACHE=/private/tmp/go-cache go test ./app/catalogapi -run 'TestPackagingModuleReplaysCachedPrepareResponse|TestPackagingModulePrepareRecordsSuccessAudit|TestPackagingPrepareRejectsInvalidJSON' -count=1`
Expected: `PASS`

- [ ] **Step 6: Commit the write-path migration**

```bash
git add app/catalogapi/handlers/governance app/catalogapi/handlers/packaging app/catalogapi/binding/packaging.go app/catalogapi/router/catalog.go app/catalogapi/router/packaging.go app/catalogapi/handlers_mutation.go app/catalogapi/handlers_packaging.go app/catalogapi/packaging_module.go app/catalogapi/packaging_access_module.go app/catalogapi/packaging_module_test.go app/catalogapi/packaging_access_module_test.go app/catalogapi/server_test.go
git commit -m "refactor: migrate governance and packaging endpoints to gin"
```

### Task 6: Remove Obsolete HTTP Glue And Simplify Makefile

**Files:**
- Modify: `Makefile`
- Delete: `app/catalogapi/routes.go`
- Delete: `app/catalogapi/middleware.go`
- Modify: `cmd/skill-registry/main.go`
- Modify: `README.md`
- Modify: `docs/local-web-catalog-e2e.zh-CN.md`
- Modify: `docs/customer-k3s-deployment.zh-CN.md`

- [ ] **Step 1: Add a failing check by comparing the retained public targets against the current help output**

Run: `make help`
Expected: help output still exists but contains too many CLI and fine-grained dev commands before cleanup

- [ ] **Step 2: Remove dead `net/http` adapter files once all route groups use gin**

```go
// delete routes.go and middleware.go after their responsibilities are covered by router/ and middleware/
```

- [ ] **Step 3: Simplify `Makefile` to retain only top-level workflows**

```make
.PHONY: help dev deploy build verify test-go test-web build-web

verify: test-go test-web

build:
	docker build -f $(CATALOG_DOCKERFILE) -t $(CATALOG_IMAGE) .
	docker build -f $(WEB_DOCKERFILE) -t $(WEB_IMAGE) .
```

- [ ] **Step 4: Fold remaining useful dev logic under `dev` and remove public CLI aliases and fine-grained kind lifecycle targets**

```make
dev:
	$(MAKE) --no-print-directory build
	$(MAKE) --no-print-directory test-go
```

- [ ] **Step 5: Update docs that mention removed make targets**

Run: `rg -n "make (cli|cli-|dev-up|dev-load|dev-apply|dev-bootstrap|dev-ready|dev-update|dev-update-apply)" README.md docs`
Expected: matches are either removed or rewritten to `make dev`, `make build`, `make verify`, or deployment entrypoints

- [ ] **Step 6: Commit the cleanup**

```bash
git add Makefile cmd/skill-registry/main.go README.md docs app/catalogapi
git commit -m "refactor: simplify backend make workflows"
```

### Task 7: Full Verification

**Files:**
- Test: `app/catalogapi`
- Test: `./...`
- Verify: `Makefile`

- [ ] **Step 1: Run the focused backend adapter suite**

Run: `env GOCACHE=/private/tmp/go-cache go test ./app/catalogapi`
Expected: `ok`

- [ ] **Step 2: Run the full Go test suite**

Run: `env GOCACHE=/private/tmp/go-cache go test ./...`
Expected: `ok`

- [ ] **Step 3: Run retained make entrypoints that do not require external cluster state**

Run: `make test-go`
Expected: `go test ./...` succeeds

Run: `make build-web`
Expected: web build succeeds

- [ ] **Step 4: Review the diff for accidental API or workflow changes**

Run: `git diff --stat HEAD~6..HEAD`
Expected: changes are limited to gin migration, handler reorganization, docs, and Makefile cleanup

- [ ] **Step 5: Commit any final doc or test adjustments from verification**

```bash
git add .
git commit -m "test: verify gin-backed catalog api refactor"
```
