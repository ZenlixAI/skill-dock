# Remove Runtime Mock And Sample Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove runtime mock-registry and sample-data boot modes from the production backend while preserving ordinary test fixtures and keeping the real catalog API and startup path working.

**Architecture:** Collapse backend startup to a single production-oriented path, delete `/mock-npm` and its backing module entirely, and remove tests that only exist for those runtime-only features. Keep generic in-memory repositories and `catalog.SampleData()` as test fixtures where they are not part of runtime behavior.

**Tech Stack:** Go 1.26, `net/http`, `github.com/gin-gonic/gin`, existing catalog/auth/packaging modules, GNU Make, repository docs

---

### Task 1: Remove Runtime Sample And Mock Startup Modes

**Files:**
- Modify: `cmd/skill-registry/main.go`
- Modify: `cmd/skill-registry/main_test.go`
- Modify: `README.md`

- [ ] **Step 1: Add failing tests that lock the new production-only startup helpers**

```go
func TestConfiguredArtifactSourceDoesNotFallbackToSampleArtifacts(t *testing.T) {
	t.Setenv("SKILL_REGISTRY_ARTIFACTS_PATH", "")
	t.Setenv("SKILL_REGISTRY_NPM_REGISTRY_URL", "")

	source := configuredArtifactSource()
	if source == nil {
		t.Fatal("expected non-nil artifact source")
	}
}
```

- [ ] **Step 2: Run the focused startup tests to verify the current code still depends on runtime sample/mock branches**

Run: `env GOCACHE=/private/tmp/go-cache go test ./cmd/skill-registry -run 'TestConfiguredArtifactSourceDoesNotFallbackToSampleArtifacts|TestConfiguredHTTPServerBuildsTimeouts' -count=1`
Expected: FAIL because `configuredArtifactSource` still depends on `sampleData` and `enableSampleData`.

- [ ] **Step 3: Simplify `main.go` so startup no longer reads sample/mock env vars or builds sample/mock dependencies**

```go
func main() {
	addr := envOrDefault("SKILL_REGISTRY_ADDR", ":8080")
	sharedWorkspaceRoot := envOrDefault("SKILL_REGISTRY_SHARED_WORKSPACE_ROOT", filepath.Join("data", "packaging-workspaces"))

	artifactSource := configuredArtifactSource()
	databaseURL := envOrDefault("SKILL_REGISTRY_DATABASE_URL", "file:"+filepath.Join("data", "skill-registry.db"))
	driverName := sharedSQLDriverNameForURL(databaseURL)

	db, err := sql.Open(driverName, databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	configureDBPool(db)

	repository, err := catalog.NewSQLRepositoryWithDB(context.Background(), db, driverName, catalog.SampleCatalogData{}, catalog.SQLRepositoryOptions{
		ArtifactSource: artifactSource,
		DisableSeed:    true,
	})
	if err != nil {
		log.Fatal(err)
	}

	server := catalogapi.NewServer(catalogapi.ServerConfig{
		Mode:                 catalogapi.ServerMode(os.Getenv("SKILL_REGISTRY_SERVER_MODE")),
		Repository:           repository,
		Packaging:            packaging.NewServiceWithWorkspaceStore(sharedWorkspaceRoot, storeSet.Workspace),
		Idempotency:          storeSet.Idempotency,
		Audit:                storeSet.Audit,
		Logger:               log.New(os.Stderr, "", 0),
		NPMRegistryURL:       os.Getenv("SKILL_REGISTRY_NPM_REGISTRY_URL"),
		PublicNPMRegistryURL: os.Getenv("SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL"),
	})
}
```

- [ ] **Step 4: Replace sample-based artifact-source fallback with explicit production-safe selection**

```go
func configuredArtifactSource() catalog.ArtifactSource {
	if path := os.Getenv("SKILL_REGISTRY_ARTIFACTS_PATH"); path != "" {
		return artifactadapter.NewJSONArtifactSource(path)
	}
	if registryURL := os.Getenv("SKILL_REGISTRY_NPM_REGISTRY_URL"); registryURL != "" {
		return artifactadapter.NewNPMRegistryArtifactSource(registryURL, nil)
	}
	return catalog.NewStaticArtifactSource(nil)
}
```

- [ ] **Step 5: Update README runtime configuration text to remove sample/mock env vars**

Run: `rg -n 'SKILL_REGISTRY_ENABLE_SAMPLE_DATA|SKILL_REGISTRY_ENABLE_MOCK_REGISTRY|mock-npm|sample data' README.md`
Expected: matches removed or rewritten to production-only configuration language.

- [ ] **Step 6: Re-run focused startup tests**

Run: `env GOCACHE=/private/tmp/go-cache go test ./cmd/skill-registry -run 'TestConfiguredArtifactSourceDoesNotFallbackToSampleArtifacts|TestConfiguredHTTPServerBuildsTimeouts|TestEnvDurationOrDefaultUsesFallbackForEmptyValue' -count=1`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add cmd/skill-registry/main.go cmd/skill-registry/main_test.go README.md
git commit -m "refactor: remove runtime sample and mock startup modes"
```

### Task 2: Delete Mock Registry HTTP Surface And Wiring

**Files:**
- Delete: `app/catalogapi/mock_registry_module.go`
- Modify: `app/catalogapi/server.go`
- Modify: `app/catalogapi/routes.go`
- Modify: `app/catalogapi/router/router.go`
- Modify: `app/catalogapi/router/system.go`
- Modify: `app/catalogapi/handlers_system.go`
- Modify: `app/catalogapi/server_workflow_seam_test.go`
- Modify: `app/catalogapi/server_test.go`

- [ ] **Step 1: Add failing tests that assert `/mock-npm` is no longer part of the server surface**

```go
func TestMockRegistryRouteIsNotRegistered(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/mock-npm/%40platform.skill%2Fpdf-skill", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}
```

- [ ] **Step 2: Run the focused catalog API tests to confirm the old mock route still exists before deletion**

Run: `env GOCACHE=/private/tmp/go-cache go test ./app/catalogapi -run 'TestMockRegistryRouteIsNotRegistered|TestServerRoutesMockRegistryThroughMockRegistryModule' -count=1`
Expected: FAIL because `/mock-npm` is still registered and the seam test still expects the mock module to run.

- [ ] **Step 3: Remove mock-registry configuration and module wiring from `ServerConfig` and `Server`**

```go
type ServerConfig struct {
	Repository                      catalog.Repository
	Mode                            ServerMode
	ReadListCacheTTL                time.Duration
	ReadDetailCacheTTL              time.Duration
	ReadMaxInFlight                 int
	WriteMaxInFlight                int
	ReaderToken                     string
	AuthoringToken                  string
	PublisherToken                  string
	NPMRegistryURL                  string
	NPMRegistryPublishAuthorization string
	NPMRegistryPublishClient        *http.Client
	PublicNPMRegistryURL            string
	AuthTokens                      map[string]auth.TokenConfig
	SessionSecret                   string
	SessionUsers                    map[string]auth.SessionUserConfig
	Packaging                       *packaging.Service
	Idempotency                     persist.IdempotencyStore
	Audit                           persist.AuditStore
	Logger                          *log.Logger
}
```

- [ ] **Step 4: Remove `/mock-npm` from the router shell and delete the dedicated handler/module**

```go
func registerSystemRoutes(engine *gin.Engine, config Config) {
	engine.Any("/healthz", config.Healthz)
	engine.Any("/readyz", config.Readyz)
	engine.Any("/metrics", config.Metrics)
}
```

- [ ] **Step 5: Delete or rewrite tests that only exercised mock-registry behavior**

Run: `rg -n '/mock-npm|MockRegistry|mockRegistryModule' app/catalogapi/server_test.go app/catalogapi/server_workflow_seam_test.go`
Expected: mock-registry-only tests and stubs removed; no production-path tests should reference the deleted route or module.

- [ ] **Step 6: Re-run focused API tests**

Run: `env GOCACHE=/private/tmp/go-cache go test ./app/catalogapi -run 'TestMockRegistryRouteIsNotRegistered|TestSkillsRequiresAuthentication|TestSessionEndpointReturnsCurrentPrincipal' -count=1`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add app/catalogapi/server.go app/catalogapi/routes.go app/catalogapi/router/router.go app/catalogapi/router/system.go app/catalogapi/handlers_system.go app/catalogapi/server_workflow_seam_test.go app/catalogapi/server_test.go
git rm app/catalogapi/mock_registry_module.go
git commit -m "refactor: remove mock registry http surface"
```

### Task 3: Delete Mock Registry Implementation And Cleanup Feature-Specific Tests

**Files:**
- Delete: `foundation/artifact/mock_registry.go`
- Modify: `app/catalogapi/server_test.go`
- Modify: `docs/local-web-catalog-e2e.zh-CN.md`
- Modify: `README.md`

- [ ] **Step 1: Add a failing repository-wide check that ensures production code no longer references the deleted runtime features**

Run: `rg -n 'SKILL_REGISTRY_ENABLE_SAMPLE_DATA|SKILL_REGISTRY_ENABLE_MOCK_REGISTRY|/mock-npm|MockRegistry' cmd app foundation README.md docs`
Expected: matches still present before cleanup.

- [ ] **Step 2: Delete the artifact-layer mock registry implementation and remove now-dead publish-flow tests that depended on it**

```bash
git rm foundation/artifact/mock_registry.go
```

- [ ] **Step 3: Rewrite docs so they no longer advertise runtime sample/mock behavior**

```md
- Docker 镜像默认关闭 sample data 与 `/mock-npm`
+ 生产部署仅支持真实 catalog 数据源与真实 npm registry 配置
```

- [ ] **Step 4: Re-run the search and verify production references are gone**

Run: `rg -n 'SKILL_REGISTRY_ENABLE_SAMPLE_DATA|SKILL_REGISTRY_ENABLE_MOCK_REGISTRY|/mock-npm|MockRegistry' cmd app foundation README.md docs`
Expected: no matches in production code or runtime-facing docs; any remaining `SampleData()` references should be test-fixture usage only.

- [ ] **Step 5: Run targeted package tests after the deletions**

Run: `env GOCACHE=/private/tmp/go-cache go test ./foundation/artifact ./cmd/skill-registry ./app/catalogapi -count=1`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add README.md docs/local-web-catalog-e2e.zh-CN.md app/catalogapi/server_test.go
git commit -m "test: remove mock-registry specific coverage and docs"
```

### Task 4: Full Verification

**Files:**
- Test: `./...`

- [ ] **Step 1: Run the full Go suite**

Run: `env GOCACHE=/private/tmp/go-cache go test ./...`
Expected: PASS

- [ ] **Step 2: Verify no production code references removed runtime mock/sample hooks**

Run: `rg -n 'SKILL_REGISTRY_ENABLE_SAMPLE_DATA|SKILL_REGISTRY_ENABLE_MOCK_REGISTRY|/mock-npm|MockRegistry' cmd app foundation`
Expected: no matches

- [ ] **Step 3: Verify remaining sample-data references are fixture-only**

Run: `rg -n 'SampleDataWithRegistryBase|SampleData\\(' business app cmd foundation`
Expected: matches only in tests or generic fixture helpers, not in production startup or production HTTP wiring.

- [ ] **Step 4: Review the final diff for unintended scope expansion**

Run: `git diff --stat HEAD~4..HEAD`
Expected: changes limited to startup wiring, mock-route removal, feature-specific test cleanup, and docs.

- [ ] **Step 5: Commit any final follow-up if verification exposed stale docs or test references**

```bash
git add .
git commit -m "test: verify production-only backend runtime cleanup"
```
