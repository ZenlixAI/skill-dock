package catalogapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/binding"
	"skill_registry/app/catalogapi/system"
	"skill_registry/business/catalog"
	"skill_registry/business/packaging"
	internalapp "skill_registry/internal/catalog/application"
	packagingapplication "skill_registry/internal/packaging/application"
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
		Repository:   catalog.NewMemoryRepository(catalog.SampleData()),
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

	mutationModule, ok := server.catalogMutationModule.(*catalogMutationModule)
	if !ok {
		t.Fatalf("expected concrete catalogMutationModule, got %T", server.catalogMutationModule)
	}
	if mutationModule.service != explicitWrite {
		t.Fatal("expected explicit catalog mutation service to win over repository provider")
	}

	if server.governanceModule.service != explicitRead {
		t.Fatal("expected explicit governance service to win over repository provider")
	}
}

func TestNewServerDoesNotConfigurePackagingPublishWorkflow(t *testing.T) {
	packagingService := &packagingApplicationStub{}

	NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
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

type readAPIModuleStub struct {
	listSkillsCalls atomic.Int64
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

func (m *readAPIModuleStub) listSkills(context.Context, auth.Principal, binding.SkillsQuery) (catalog.SkillsPage, *catalog.Error) {
	m.listSkillsCalls.Add(1)
	return catalog.SkillsPage{}, nil
}

func (m *readAPIModuleStub) listSkillsCallCount() int {
	return int(m.listSkillsCalls.Load())
}

type systemModuleStub struct {
	readyzCalls atomic.Int64
}

func (m *systemModuleStub) Readyz(context.Context) (system.Readyz, int) {
	m.readyzCalls.Add(1)
	return system.Readyz{
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
