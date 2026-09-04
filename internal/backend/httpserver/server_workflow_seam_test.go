package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	catalog "skill_registry/internal/catalog/domain"
	cataloginfra "skill_registry/internal/catalog/infra"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
	internalsystemhttp "skill_registry/internal/system/http"
)

func TestServerRoutesAuditEventsThroughAuditQueryModule(t *testing.T) {
	module := newAuditQueryModuleStubForTest(t)
	server := newTestServer()
	server.auditListEvents = module.ListEvents

	request := httptest.NewRequest(http.MethodGet, "/api/v1/audit-events", nil)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if got := module.listEventsCallCount(); got != 1 {
		t.Fatalf("expected audit query module to be called once, got %d", got)
	}
}

func TestServerWorkflowRoutesSkillsThroughCatalogReadModule(t *testing.T) {
	module := newCatalogReadModuleStubForTest(t)
	server := newTestServerWithCatalogReadModule(t, module)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills?q=pdf", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if got := module.listSkillsCallCount(); got != 1 {
		t.Fatalf("expected catalog read module to be called once, got %d", got)
	}
}

func TestServerReadModeDoesNotRegisterPackagingRoutes(t *testing.T) {
	server := newTestServer()
	server.mode = ServerModeRead

	request := httptest.NewRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", strings.NewReader(`{}`))
	request.Header.Set("X-Role", "publisher")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
}

func TestServerWriteModeDoesNotRegisterCatalogListRoutes(t *testing.T) {
	server := newTestServer()
	server.mode = ServerModeWrite

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/search?q=pdf", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
}

func TestNewServerUsesInternalPlatformCacheAndLimiterTypes(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository: cataloginfra.NewMemoryRepository(cataloginfra.SampleData()),
	})

	if _, ok := any(server.readCache).(*platformhttp.ReadCache); !ok {
		t.Fatalf("expected readCache to use internal platform cache, got %T", server.readCache)
	}
	if _, ok := any(server.readLimiter).(*platformhttp.RequestLimiter); !ok {
		t.Fatalf("expected readLimiter to use internal platform limiter, got %T", server.readLimiter)
	}
	if _, ok := any(server.writeLimiter).(*platformhttp.RequestLimiter); !ok {
		t.Fatalf("expected writeLimiter to use internal platform limiter, got %T", server.writeLimiter)
	}
}

func TestServerReadModeDoesNotServeReleaseRecordMutation(t *testing.T) {
	server := newTestServer()
	server.mode = ServerModeRead

	request := httptest.NewRequest(http.MethodPut, "/api/v1/skills/%40platform.skill%2Fpdf-skill/versions/1.2.0/release-record", strings.NewReader(`{}`))
	request.Header.Set("X-Role", "publisher")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
}

func TestServerRoutesReadyzThroughSystemModule(t *testing.T) {
	module := &systemModuleStub{}
	server := newTestServer()
	server.systemReadyz = module.Readyz

	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if got := module.readyzCallCount(); got != 1 {
		t.Fatalf("expected system module to be called once, got %d", got)
	}
}

func TestServerRoutesReleaseRecordThroughCatalogMutationWorkflow(t *testing.T) {
	module := newCatalogMutationWorkflowStubForTest(t)
	server := newTestServer()
	server.governanceWorkflow = governanceWorkflowAdapter{stub: module}

	request := httptest.NewRequest(http.MethodPut, "/api/v1/skills/%40platform.skill%2Fpdf-skill/versions/1.2.0/release-record", strings.NewReader(`{}`))
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if got := module.createReleaseRecordCallCount(); got != 1 {
		t.Fatalf("expected mutation module to be called once, got %d", got)
	}
}

func TestServerRoutesSkillGovernanceThroughCatalogMutationWorkflow(t *testing.T) {
	module := newCatalogMutationWorkflowStubForTest(t)
	server := newTestServer()
	server.governanceWorkflow = governanceWorkflowAdapter{stub: module}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/skills/%40platform.skill%2Fcode-review/governance-changes", strings.NewReader(`{"action":"retire","reason":"merged"}`))
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if got := module.applySkillGovernanceCallCount(); got != 1 {
		t.Fatalf("expected skill governance workflow to be called once, got %d", got)
	}
}

func TestServerRoutesVersionGovernanceThroughCatalogMutationWorkflow(t *testing.T) {
	module := newCatalogMutationWorkflowStubForTest(t)
	server := newTestServer()
	server.governanceWorkflow = governanceWorkflowAdapter{stub: module}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/skills/%40platform.skill%2Fpdf-skill/versions/1.2.0/governance-changes", strings.NewReader(`{"action":"yank","reason":"security issue"}`))
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if got := module.applyVersionGovernanceCallCount(); got != 1 {
		t.Fatalf("expected version governance workflow to be called once, got %d", got)
	}
}

func TestServerLegacyPackagingWorkspaceRoutesReturnNotFound(t *testing.T) {
	server := newTestServer()

	requests := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodPost, path: "/api/v1/skill-packaging/prepare", body: `{}`},
		{method: http.MethodPost, path: "/api/v1/skill-packaging/validate", body: `{}`},
		{method: http.MethodPost, path: "/api/v1/skill-packaging/pack", body: `{}`},
		{method: http.MethodPost, path: "/api/v1/skill-packaging/cleanup", body: `{}`},
		{method: http.MethodGet, path: "/api/v1/skill-packaging/bundle?workspaceHandle=ws-legacy&packageName=%40platform.skill%2Fpdf-skill&version=1.2.0"},
	}

	for _, tc := range requests {
		request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		request.Header.Set("X-Role", "publisher")
		if tc.method == http.MethodPost {
			request.Header.Set("Content-Type", "application/json")
		}
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, request)

		if recorder.Code != http.StatusNotFound {
			t.Fatalf("expected %s %s to return 404, got %d body=%s", tc.method, tc.path, recorder.Code, recorder.Body.String())
		}
	}
}

type governanceWorkflowAdapter struct {
	stub *catalogMutationWorkflowStub
}

func (a governanceWorkflowAdapter) ListGovernanceChanges(ctx context.Context, principal auth.Principal, query catalog.GovernanceQuery) (catalog.GovernancePage, *catalog.Error) {
	return a.stub.ListGovernanceChanges(ctx, principal, query)
}

func (a governanceWorkflowAdapter) CreateReleaseRecord(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error) {
	return a.stub.createReleaseRecord(ctx, principal, packageName, versionPath, rawBody)
}

func (a governanceWorkflowAdapter) ApplySkillGovernanceChange(ctx context.Context, principal auth.Principal, packageName string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
	return a.stub.applySkillGovernanceChange(ctx, principal, packageName, rawBody)
}

func (a governanceWorkflowAdapter) ApplyVersionGovernanceChange(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
	return a.stub.applyVersionGovernanceChange(ctx, principal, packageName, versionPath, rawBody)
}

type systemModuleStub struct {
	readyzCalls atomic.Int64
}

func (m *systemModuleStub) Readyz(context.Context) (internalsystemhttp.Readyz, int) {
	m.readyzCalls.Add(1)
	return internalsystemhttp.Readyz{
		Status: "ready",
		Checks: map[string]string{
			"repository":  "ok",
			"audit":       "ok",
			"idempotency": "ok",
		},
	}, http.StatusOK
}

func (m *systemModuleStub) readyzCallCount() int {
	return int(m.readyzCalls.Load())
}
