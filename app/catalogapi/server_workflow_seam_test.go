package catalogapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"skill_registry/app/auth"
	"skill_registry/business/catalog"
)

func TestServerRoutesAuditEventsThroughAuditQueryModule(t *testing.T) {
	module := newAuditQueryModuleStubForTest(t)
	server := newTestServer()
	server.auditQueryModule = module

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

func TestServerRoutesSkillsThroughReadAPIModule(t *testing.T) {
	module := &readAPIModuleStub{}
	server := newTestServer()
	server.readAPIModule = module

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills?q=pdf", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if got := module.listSkillsCallCount(); got != 1 {
		t.Fatalf("expected read API module to be called once, got %d", got)
	}
}

func TestServerRoutesReadyzThroughSystemModule(t *testing.T) {
	module := &systemModuleStub{}
	server := newTestServer()
	server.systemModule = module

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

func TestServerRoutesReleaseRecordThroughCatalogMutationModule(t *testing.T) {
	module := newCatalogMutationModuleStubForTest(t)
	server := newTestServer()
	server.catalogMutationModule = module

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

func TestServerRoutesSkillGovernanceThroughCatalogMutationModule(t *testing.T) {
	module := newCatalogMutationModuleStubForTest(t)
	server := newTestServer()
	server.catalogMutationModule = module

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

func TestServerRoutesVersionGovernanceThroughCatalogMutationModule(t *testing.T) {
	module := newCatalogMutationModuleStubForTest(t)
	server := newTestServer()
	server.catalogMutationModule = module

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

func TestServerRoutesBundleDownloadThroughPackagingAccessModule(t *testing.T) {
	module := newPackagingAccessModuleStubForTest(t)
	server := newTestServer()
	server.packagingAccessModule = module

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skill-packaging/bundle?workspacePath=/tmp/workspace&packageName=%40platform.skill%2Fpdf-skill&version=1.2.0", nil)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if got := module.downloadBundleCallCount(); got != 1 {
		t.Fatalf("expected packaging access module to be called once, got %d", got)
	}
}

func TestServerRoutesPackagingCleanupThroughPackagingAccessModule(t *testing.T) {
	module := newPackagingAccessModuleStubForTest(t)
	server := newTestServer()
	server.packagingAccessModule = module

	request := httptest.NewRequest(http.MethodPost, "/api/v1/skill-packaging/cleanup", strings.NewReader(`{"workspacePath":"/tmp/workspace"}`))
	request.Header.Set("X-Role", "publisher")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if got := module.cleanupCallCount(); got != 1 {
		t.Fatalf("expected cleanup workflow to be called once, got %d", got)
	}
}

func TestServerSessionDeniedPathUsesSharedAccessWorkflow(t *testing.T) {
	workflow := newAccessWorkflowStubForTest(t, false)
	server := newTestServer()
	server.accessWorkflow = workflow

	request := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusForbidden, recorder.Code, recorder.Body.String())
	}
	if got := workflow.denyCallCount(); got != 1 {
		t.Fatalf("expected shared access workflow to deny once, got %d", got)
	}
}

func TestServerRoutesMockRegistryThroughMockRegistryModule(t *testing.T) {
	module := &mockRegistryModuleStub{}
	server := newTestServer()
	server.mockRegistryModule = module

	request := httptest.NewRequest(http.MethodGet, "/mock-npm/%40platform.skill%2Fpdf-skill", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if got := module.handleCallCount(); got != 1 {
		t.Fatalf("expected mock registry module to be called once, got %d", got)
	}
}

type readAPIModuleStub struct {
	listSkillsCalls atomic.Int64
}

func (m *readAPIModuleStub) listSkills(context.Context, auth.Principal, url.Values) (catalog.SkillsPage, *catalog.Error) {
	m.listSkillsCalls.Add(1)
	return catalog.SkillsPage{}, nil
}

func (m *readAPIModuleStub) listSkillsCallCount() int {
	return int(m.listSkillsCalls.Load())
}

type systemModuleStub struct {
	readyzCalls atomic.Int64
}

func (m *systemModuleStub) readyz(context.Context) (readyzResponse, int) {
	m.readyzCalls.Add(1)
	return readyzResponse{
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

type mockRegistryModuleStub struct {
	handleCalls atomic.Int64
}

func (m *mockRegistryModuleStub) handleRequest(writer http.ResponseWriter, _ *http.Request) {
	m.handleCalls.Add(1)
	writer.WriteHeader(http.StatusOK)
}

func (m *mockRegistryModuleStub) handleCallCount() int {
	return int(m.handleCalls.Load())
}
