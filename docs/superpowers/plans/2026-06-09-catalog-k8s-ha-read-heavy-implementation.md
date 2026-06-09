# Catalog K8s HA Read-Heavy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the current `catalog` backend into read-optimized and write-optimized Kubernetes workloads while adding graceful shutdown, explicit DB pool tuning, bounded read/write concurrency, and Helm support for the new topology.

**Architecture:** Keep a single Go binary and add an explicit runtime mode that gates route registration and runtime dependencies. Deploy that binary as `catalog-read` and `catalog-write` via Helm, with short-lived read-side in-memory caches, write-side shared RWX workspace storage, and Kubernetes lifecycle/HA policies tuned per workload.

**Tech Stack:** Go, `net/http`, `database/sql`, PostgreSQL via `pgx`, in-memory caches, Helm, Traefik, Kubernetes probes/HPA/PDB/topology spread.

---

## File Structure

### Runtime and server mode

- Modify: `cmd/skill-registry/main.go`
  - Parse server mode, DB pool env, graceful shutdown config, and shared workspace root config.
- Modify: `app/catalogapi/server.go`
  - Add server mode-aware route registration and expose a mode-aware `http.Handler`.
- Create: `app/catalogapi/server_mode.go`
  - Define mode enum, validation, and route-surface decisions.
- Create: `app/catalogapi/server_lifecycle.go`
  - Build configured `http.Server` and graceful shutdown helpers.

### Read-side performance controls

- Create: `app/catalogapi/read_cache.go`
  - TTL cache for read responses.
- Create: `app/catalogapi/read_limiter.go`
  - Bounded concurrency gate for expensive read handlers.
- Modify: `app/catalogapi/catalog_read_module.go`
  - Use limiter and cache hooks around list/detail reads.
- Modify: `app/catalogapi/read_api_module.go`
  - Reuse normalized cache semantics for read endpoints.

### Write-side protection

- Create: `app/catalogapi/write_limiter.go`
  - Bounded concurrency gate for packaging and publish handlers.
- Modify: `app/catalogapi/packaging_module.go`
  - Enforce write-side concurrency policy.
- Modify: `app/catalogapi/packaging_publish.go`
  - Reuse the same gate for publish paths.

### Tests

- Modify: `app/catalogapi/server_test.go`
  - Add route-surface, lifecycle, and mode tests.
- Modify: `app/catalogapi/server_workflow_seam_test.go`
  - Verify read/write routing through the right modules in each mode.
- Create: `app/catalogapi/read_cache_test.go`
  - Verify TTL, key scoping, and non-caching of errors.
- Create: `app/catalogapi/read_limiter_test.go`
  - Verify overload behavior.
- Create: `app/catalogapi/write_limiter_test.go`
  - Verify bounded write concurrency.
- Modify: `cmd/skill-registry/main_test.go`
  - Add env/config parsing tests if the package already has or needs main helpers under test.

### Helm chart

- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/values.yaml`
  - Add `catalogRead` and `catalogWrite` values.
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/values.schema.json`
  - Validate the new values structure.
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/_helpers.tpl`
  - Add workload-specific naming helpers and shared env helpers.
- Create: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/catalog-read/deployment.yaml`
- Create: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/catalog-read/service.yaml`
- Create: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/catalog-read/hpa.yaml`
- Create: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/catalog-read/pdb.yaml`
- Create: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/catalog-write/deployment.yaml`
- Create: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/catalog-write/service.yaml`
- Create: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/catalog-write/pdb.yaml`
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/traefik/web-ingressroute.yaml`
  - Route read endpoints to read service and packaging/mutation paths to write service.
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/README.md`
  - Document deployment shape and required RWX configuration.

## Task 1: Add Server Mode and Route Gating

**Files:**
- Create: `app/catalogapi/server_mode.go`
- Modify: `app/catalogapi/server.go`
- Modify: `app/catalogapi/server_test.go`
- Test: `app/catalogapi/server_test.go`

- [ ] **Step 1: Write the failing route-surface tests**

```go
func TestServerReadModeDoesNotRegisterPackagingRoutes(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		Mode:       ServerModeRead,
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", strings.NewReader(`{}`))
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
}

func TestServerWriteModeDoesNotRegisterCatalogReadRoutes(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		Mode:       ServerModeWrite,
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./app/catalogapi -run 'TestServerReadModeDoesNotRegisterPackagingRoutes|TestServerWriteModeDoesNotRegisterCatalogReadRoutes'`
Expected: FAIL because `ServerConfig` has no mode and all routes are still registered.

- [ ] **Step 3: Write the minimal mode implementation**

```go
// app/catalogapi/server_mode.go
package catalogapi

type ServerMode string

const (
	ServerModeAll   ServerMode = "all"
	ServerModeRead  ServerMode = "read"
	ServerModeWrite ServerMode = "write"
)

func normalizeServerMode(raw ServerMode) ServerMode {
	switch raw {
	case ServerModeRead, ServerModeWrite:
		return raw
	default:
		return ServerModeAll
	}
}

func (m ServerMode) servesReadRoutes() bool {
	return m == ServerModeAll || m == ServerModeRead
}

func (m ServerMode) servesWriteRoutes() bool {
	return m == ServerModeAll || m == ServerModeWrite
}
```

```go
// inside app/catalogapi/server.go
type ServerConfig struct {
	// existing fields...
	Mode ServerMode
}

type Server struct {
	// existing fields...
	mode ServerMode
}

func NewServer(config ServerConfig) *Server {
	mode := normalizeServerMode(config.Mode)
	server := &Server{
		// existing fields...
		mode: mode,
	}
	// existing module wiring...
	return server
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", http.HandlerFunc(s.handleHealthz))
	mux.Handle("/readyz", http.HandlerFunc(s.handleReadyz))
	mux.Handle("/metrics", http.HandlerFunc(s.handleMetrics))

	if s.mode.servesReadRoutes() {
		mux.Handle("/api/v1/skills", s.requireAction(s.handleSkills, auth.ActionReadCatalog))
		mux.Handle("/api/v1/skills/search", s.requireAction(s.handleSkills, auth.ActionReadCatalog))
		mux.Handle("/api/v1/skills/curated", s.requireAction(s.handleCuratedSkills, auth.ActionReadCatalog))
		mux.Handle("/api/v1/governance-changes", s.requireAction(s.handleGovernanceChanges, auth.ActionReadCatalog))
		mux.Handle("/api/v1/skills/", s.requireAction(s.handleSkillSubtree, auth.ActionReadCatalog))
	}

	if s.mode.servesWriteRoutes() {
		mux.Handle("/api/v1/skill-packaging/prepare", s.requireAction(s.handlePackagingPrepare, auth.ActionPackagingPrepare))
		mux.Handle("/api/v1/skill-packaging/validate", s.requireAction(s.handlePackagingValidate, auth.ActionPackagingValidate))
		mux.Handle("/api/v1/skill-packaging/pack", s.requireAction(s.handlePackagingPack, auth.ActionPackagingPack))
		mux.Handle("/api/v1/skill-packaging/publish", s.requireAction(s.handlePackagingPublish, auth.ActionPackagingPublish))
		mux.Handle("/api/v1/skill-packaging/cleanup", s.requireAction(s.handlePackagingCleanup, auth.ActionPackagingPack))
		mux.Handle("/api/v1/skill-packaging/bundle", s.requireAction(s.handlePackagingBundleDownload, auth.ActionPackagingPack))
	}

	return s.withCORS(s.withRequestMetadata(s.withObservability(mux)))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./app/catalogapi -run 'TestServerReadModeDoesNotRegisterPackagingRoutes|TestServerWriteModeDoesNotRegisterCatalogReadRoutes'`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add app/catalogapi/server.go app/catalogapi/server_mode.go app/catalogapi/server_test.go
git commit -m "feat: add catalog server modes"
```

## Task 2: Add Graceful Shutdown and DB Pool Configuration

**Files:**
- Modify: `cmd/skill-registry/main.go`
- Create: `app/catalogapi/server_lifecycle.go`
- Modify: `app/catalogapi/server_test.go`
- Test: `app/catalogapi/server_test.go`

- [ ] **Step 1: Write the failing lifecycle/config tests**

```go
func TestConfiguredServerBuildsHTTPServerWithTimeouts(t *testing.T) {
	server := NewServer(ServerConfig{Repository: catalog.NewMemoryRepository(catalog.SampleData())})
	httpServer := newHTTPServer(":8080", server.Handler(), HTTPServerConfig{
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	})

	if httpServer.ReadHeaderTimeout != 2*time.Second {
		t.Fatalf("expected read header timeout, got %v", httpServer.ReadHeaderTimeout)
	}
	if httpServer.IdleTimeout != 30*time.Second {
		t.Fatalf("expected idle timeout, got %v", httpServer.IdleTimeout)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./app/catalogapi -run TestConfiguredServerBuildsHTTPServerWithTimeouts`
Expected: FAIL because lifecycle helper does not exist.

- [ ] **Step 3: Implement minimal lifecycle and pool config support**

```go
// app/catalogapi/server_lifecycle.go
package catalogapi

import (
	"net/http"
	"time"
)

type HTTPServerConfig struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func newHTTPServer(addr string, handler http.Handler, cfg HTTPServerConfig) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}
}
```

```go
// inside cmd/skill-registry/main.go
db.SetMaxOpenConns(envIntOrDefault("SKILL_REGISTRY_DB_MAX_OPEN_CONNS", 20))
db.SetMaxIdleConns(envIntOrDefault("SKILL_REGISTRY_DB_MAX_IDLE_CONNS", 20))
db.SetConnMaxLifetime(envDurationOrDefault("SKILL_REGISTRY_DB_CONN_MAX_LIFETIME", time.Hour))
db.SetConnMaxIdleTime(envDurationOrDefault("SKILL_REGISTRY_DB_CONN_MAX_IDLE_TIME", 15*time.Minute))

httpServer := catalogapi.NewHTTPServer(addr, server.Handler(), catalogapi.HTTPServerConfig{
	ReadHeaderTimeout: envDurationOrDefault("SKILL_REGISTRY_HTTP_READ_HEADER_TIMEOUT", 2*time.Second),
	ReadTimeout:       envDurationOrDefault("SKILL_REGISTRY_HTTP_READ_TIMEOUT", 15*time.Second),
	WriteTimeout:      envDurationOrDefault("SKILL_REGISTRY_HTTP_WRITE_TIMEOUT", 60*time.Second),
	IdleTimeout:       envDurationOrDefault("SKILL_REGISTRY_HTTP_IDLE_TIMEOUT", 60*time.Second),
})
```

- [ ] **Step 4: Run focused tests**

Run: `go test ./app/catalogapi -run TestConfiguredServerBuildsHTTPServerWithTimeouts`
Expected: PASS

Run: `go test ./cmd/skill-registry`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/skill-registry/main.go app/catalogapi/server_lifecycle.go app/catalogapi/server_test.go
git commit -m "feat: add catalog lifecycle and db pool config"
```

## Task 3: Add Read-Side TTL Cache

**Files:**
- Create: `app/catalogapi/read_cache.go`
- Create: `app/catalogapi/read_cache_test.go`
- Modify: `app/catalogapi/catalog_read_module.go`
- Modify: `app/catalogapi/read_api_module.go`

- [ ] **Step 1: Write the failing cache tests**

```go
func TestReadCacheExpiresEntriesAfterTTL(t *testing.T) {
	cache := newReadCache(20 * time.Millisecond)
	cache.store("skills?query=pdf", []byte(`{"items":[1]}`))

	if _, ok := cache.lookup("skills?query=pdf"); !ok {
		t.Fatal("expected cache hit before ttl")
	}

	time.Sleep(30 * time.Millisecond)
	if _, ok := cache.lookup("skills?query=pdf"); ok {
		t.Fatal("expected cache miss after ttl")
	}
}

func TestReadCacheScopesByPrincipalKey(t *testing.T) {
	cache := newReadCache(time.Second)
	cache.store("reader\x00@company.skill/a\x00skills?query=pdf", []byte(`{"items":[1]}`))

	if _, ok := cache.lookup("reader\x00@company.skill/b\x00skills?query=pdf"); ok {
		t.Fatal("expected scoped cache miss")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./app/catalogapi -run 'TestReadCacheExpiresEntriesAfterTTL|TestReadCacheScopesByPrincipalKey'`
Expected: FAIL because read cache does not exist.

- [ ] **Step 3: Implement minimal cache**

```go
package catalogapi

type readCache struct {
	ttl     time.Duration
	mu      sync.RWMutex
	entries map[string]readCacheEntry
}

type readCacheEntry struct {
	body      []byte
	expiresAt time.Time
}
```

```go
func newReadCache(ttl time.Duration) *readCache {
	return &readCache{ttl: ttl, entries: map[string]readCacheEntry{}}
}

func (c *readCache) lookup(key string) ([]byte, bool) {
	// clone body and expire stale entries
}

func (c *readCache) store(key string, body []byte) {
	// copy body and save expiry
}
```

- [ ] **Step 4: Wire cache only around successful read handlers**

```go
// inside catalog_read_module.go or read_api_module.go helper path
if cached, ok := m.cache.lookup(cacheKey); ok {
	return json.RawMessage(cached), nil
}

response, err := m.repo.ListSkills(ctx, query)
if err == nil {
	body, marshalErr := json.Marshal(response)
	if marshalErr == nil {
		m.cache.store(cacheKey, body)
	}
}
```

- [ ] **Step 5: Run focused tests**

Run: `go test ./app/catalogapi -run 'TestReadCacheExpiresEntriesAfterTTL|TestReadCacheScopesByPrincipalKey'`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add app/catalogapi/read_cache.go app/catalogapi/read_cache_test.go app/catalogapi/catalog_read_module.go app/catalogapi/read_api_module.go
git commit -m "feat: add read-side ttl cache"
```

## Task 4: Add Read and Write Concurrency Gates

**Files:**
- Create: `app/catalogapi/read_limiter.go`
- Create: `app/catalogapi/read_limiter_test.go`
- Create: `app/catalogapi/write_limiter.go`
- Create: `app/catalogapi/write_limiter_test.go`
- Modify: `app/catalogapi/catalog_read_module.go`
- Modify: `app/catalogapi/packaging_module.go`
- Modify: `app/catalogapi/packaging_publish.go`

- [ ] **Step 1: Write the failing limiter tests**

```go
func TestReadLimiterRejectsWhenCapacityIsExceeded(t *testing.T) {
	limiter := newRequestLimiter(1)
	release, err := limiter.acquire()
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	defer release()

	_, err = limiter.acquire()
	if err == nil {
		t.Fatal("expected limiter rejection")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./app/catalogapi -run TestReadLimiterRejectsWhenCapacityIsExceeded`
Expected: FAIL because limiter does not exist.

- [ ] **Step 3: Implement minimal bounded limiter**

```go
type requestLimiter struct {
	tokens chan struct{}
}

func newRequestLimiter(limit int) *requestLimiter {
	if limit <= 0 {
		limit = 1
	}
	return &requestLimiter{tokens: make(chan struct{}, limit)}
}

func (l *requestLimiter) acquire() (func(), error) {
	select {
	case l.tokens <- struct{}{}:
		return func() { <-l.tokens }, nil
	default:
		return nil, catalog.NewError(http.StatusTooManyRequests, "request_limit_exceeded", "Request concurrency limit exceeded.", nil)
	}
}
```

- [ ] **Step 4: Apply the limiter to read and write paths**

```go
release, err := m.readLimiter.acquire()
if err != nil {
	return catalog.SkillsPage{}, catalogErrorFromError(err)
}
defer release()
```

```go
release, err := m.writeLimiter.acquire()
if err != nil {
	return nil, err.Status, err
}
defer release()
```

- [ ] **Step 5: Run focused tests**

Run: `go test ./app/catalogapi -run 'TestReadLimiterRejectsWhenCapacityIsExceeded|TestWriteLimiterRejectsWhenCapacityIsExceeded'`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add app/catalogapi/read_limiter.go app/catalogapi/read_limiter_test.go app/catalogapi/write_limiter.go app/catalogapi/write_limiter_test.go app/catalogapi/catalog_read_module.go app/catalogapi/packaging_module.go app/catalogapi/packaging_publish.go
git commit -m "feat: bound catalog read and write concurrency"
```

## Task 5: Add Helm Read/Write Workloads

**Files:**
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/values.yaml`
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/values.schema.json`
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/_helpers.tpl`
- Create: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/catalog-read/deployment.yaml`
- Create: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/catalog-read/service.yaml`
- Create: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/catalog-read/hpa.yaml`
- Create: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/catalog-read/pdb.yaml`
- Create: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/catalog-write/deployment.yaml`
- Create: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/catalog-write/service.yaml`
- Create: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/catalog-write/pdb.yaml`
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/templates/traefik/web-ingressroute.yaml`
- Modify: `/Users/ultraman/workspace/skill-registry-helm-chart/README.md`

- [ ] **Step 1: Write the failing Helm template expectations**

```bash
helm template skill-registry /Users/ultraman/workspace/skill-registry-helm-chart \
  --set catalogRead.replicaCount=3 \
  --set catalogWrite.replicaCount=1
```

Expected: FAIL or missing resources because `catalogRead` and `catalogWrite` templates do not exist.

- [ ] **Step 2: Add values structure**

```yaml
catalogRead:
  replicaCount: 3
  service:
    port: 8080
  cache:
    listTTLSeconds: 2
    detailTTLSeconds: 5
  concurrency:
    maxInFlight: 64
  dbPool:
    maxOpenConns: 40
    maxIdleConns: 40
  hpa:
    enabled: true
    minReplicas: 3
    maxReplicas: 10
  pdb:
    enabled: true
    minAvailable: 2

catalogWrite:
  replicaCount: 1
  service:
    port: 8080
  concurrency:
    maxInFlight: 4
  dbPool:
    maxOpenConns: 10
    maxIdleConns: 10
  workspace:
    enabled: true
    mountPath: /srv/packaging-workspaces
    existingClaim: ""
```

- [ ] **Step 3: Render separate deployments**

```yaml
env:
  - name: SKILL_REGISTRY_SERVER_MODE
    value: read
  - name: SKILL_REGISTRY_DB_MAX_OPEN_CONNS
    value: {{ .Values.catalogRead.dbPool.maxOpenConns | quote }}
  - name: SKILL_REGISTRY_READ_CACHE_LIST_TTL
    value: {{ .Values.catalogRead.cache.listTTLSeconds | quote }}
```

```yaml
env:
  - name: SKILL_REGISTRY_SERVER_MODE
    value: write
  - name: SKILL_REGISTRY_SHARED_WORKSPACE_ROOT
    value: {{ .Values.catalogWrite.workspace.mountPath | quote }}
```

- [ ] **Step 4: Add path routing and HA resources**

```yaml
# read hpa
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
```

```yaml
# read pdb
apiVersion: policy/v1
kind: PodDisruptionBudget
```

```yaml
# ingress route concept
- match: Host(`skills.example.com`) && (PathPrefix(`/api/v1/skills`) || PathPrefix(`/api/v1/governance-changes`))
  services:
    - name: {{ include "skill-registry.catalogReadName" . }}
```

- [ ] **Step 5: Run Helm verification**

Run: `helm template skill-registry /Users/ultraman/workspace/skill-registry-helm-chart > /tmp/skill-registry-rendered.yaml`
Expected: renders both catalog workloads, read HPA/PDB, write PDB, and separate services.

- [ ] **Step 6: Commit**

```bash
git add /Users/ultraman/workspace/skill-registry-helm-chart
git commit -m "feat: split helm catalog read and write workloads"
```

## Task 6: Full Verification

**Files:**
- Modify as needed from prior tasks
- Test: all touched Go packages and Helm rendering

- [ ] **Step 1: Run focused Go tests**

Run: `go test ./app/catalogapi ./cmd/skill-registry`
Expected: PASS

- [ ] **Step 2: Run full Go test suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 3: Run CLI regression suite**

Run: `cd apps/skills-cli && npm test`
Expected: PASS

- [ ] **Step 4: Run Helm render verification**

Run: `helm template skill-registry /Users/ultraman/workspace/skill-registry-helm-chart > /tmp/skill-registry-rendered.yaml`
Expected: PASS

- [ ] **Step 5: Commit final polish if needed**

```bash
git add docs/superpowers/specs/2026-06-09-catalog-k8s-ha-read-heavy-design.md docs/superpowers/plans/2026-06-09-catalog-k8s-ha-read-heavy-implementation.md
git commit -m "docs: record catalog ha implementation plan"
```

## Self-Review

### Spec coverage

- server mode and route gating: Task 1
- graceful shutdown and pool tuning: Task 2
- read-side eventual consistency cache: Task 3
- read/write concurrency protection: Task 4
- Helm split and RWX write storage: Task 5
- verification requirements: Task 6

No spec requirement is intentionally omitted for phase 1.

### Placeholder scan

- No `TODO`
- No `TBD`
- No unresolved names that are not introduced in the plan

### Type consistency

- `ServerMode` is used consistently as `read|write|all`
- concurrency gate uses a shared `requestLimiter` shape
- env var naming is consistent across runtime and Helm examples

