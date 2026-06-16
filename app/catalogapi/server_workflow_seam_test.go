package catalogapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	internalapp "skill_registry/internal/catalog/application"
	cataloginfra "skill_registry/internal/catalog/infra"
	packagingapplication "skill_registry/internal/packaging/application"
	packaging "skill_registry/internal/packaging/domain"
	internalsystemhttp "skill_registry/internal/system/http"
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

func TestNewServerUsesExplicitWorkflowServicesWhenProvided(t *testing.T) {
	providerRead := internalapp.NewReadService(serverQueryStoreStub{})
	providerWrite := internalapp.NewWriteService(serverAggregateStateStoreStub{}, serverArtifactSourceStub{})
	explicitRead := internalapp.NewReadService(serverQueryStoreStub{})
	explicitWrite := internalapp.NewWriteService(serverAggregateStateStoreStub{}, serverArtifactSourceStub{})
	provider := &repositoryApplicationProviderStub{
		Repository:   cataloginfra.NewMemoryRepository(cataloginfra.SampleData()),
		readService:  providerRead,
		writeService: providerWrite,
	}

	server := NewServer(ServerConfig{
		Repository:               provider,
		CatalogReadService:       explicitRead,
		CatalogMutationService:   explicitWrite,
		CatalogGovernanceService: explicitRead,
	})

	readModule, ok := server.catalogReadModule.(*catalogReadModule)
	if !ok {
		t.Fatalf("expected concrete catalogReadModule, got %T", server.catalogReadModule)
	}
	if readModule.service != explicitRead {
		t.Fatal("expected explicit catalog read service to win over repository provider")
	}

	if server.catalogMutationService != explicitWrite {
		t.Fatal("expected explicit catalog mutation service to win over repository provider")
	}

	if server.catalogGovernanceService != explicitRead {
		t.Fatal("expected explicit governance service to win over repository provider")
	}
}

func TestNewServerDoesNotConfigurePackagingPublishWorkflow(t *testing.T) {
	packagingService := &packagingApplicationStub{}

	NewServer(ServerConfig{
		Repository: cataloginfra.NewMemoryRepository(cataloginfra.SampleData()),
		Packaging:  packagingService,
	})

	if packagingService.configurePublishWorkflowCalls != 0 {
		t.Fatalf("expected server construction not to configure packaging publish workflow, got %d calls", packagingService.configurePublishWorkflowCalls)
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

func TestServerRoutesReleaseRecordThroughCatalogMutationWorkflow(t *testing.T) {
	module := newCatalogMutationWorkflowStubForTest(t)
	server := newTestServer()
	server.createReleaseRecordFunc = module.createReleaseRecord

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
	server.applySkillGovernanceChangeFunc = module.applySkillGovernanceChange

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
	server.applyVersionGovernanceChangeFunc = module.applyVersionGovernanceChange

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

func TestServerRoutesBundleDownloadThroughPackagingAccessWorkflow(t *testing.T) {
	module := newPackagingAccessWorkflowStubForTest(t)
	server := newTestServer()
	server.downloadBundleFunc = module.downloadBundle

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

func TestServerRoutesPackagingCleanupThroughPackagingAccessWorkflow(t *testing.T) {
	module := newPackagingAccessWorkflowStubForTest(t)
	server := newTestServer()
	server.cleanupPackagingFunc = module.cleanup

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

type packagingApplicationStub struct {
	configurePublishWorkflowCalls int
}

func (s *packagingApplicationStub) PrepareContext(context.Context, packaging.PrepareRequest) (packaging.PrepareResponse, error) {
	return packaging.PrepareResponse{}, nil
}

func (s *packagingApplicationStub) ValidateContext(context.Context, packaging.ValidateRequest) (packaging.ValidateResponse, error) {
	return packaging.ValidateResponse{}, nil
}

func (s *packagingApplicationStub) PackContext(context.Context, packaging.PackRequest) (packaging.PackResponse, error) {
	return packaging.PackResponse{}, nil
}

func (s *packagingApplicationStub) CleanupContext(context.Context, string) error {
	return nil
}

func (s *packagingApplicationStub) BundleFilePathContext(context.Context, string, packaging.Target) (string, error) {
	return "", nil
}

func (s *packagingApplicationStub) PublishContext(context.Context, packaging.PublishRequest) (packaging.PublishResponse, error) {
	return packaging.PublishResponse{}, nil
}

func (s *packagingApplicationStub) ConfigurePublishWorkflow(packagingapplication.ReleaseRecordCreator, packagingapplication.PackagePublisher) {
	s.configurePublishWorkflowCalls++
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
