package catalogapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
