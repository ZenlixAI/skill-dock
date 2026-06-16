package catalogapi

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	catalog "skill_registry/internal/catalog/domain"
	internalcataloghttp "skill_registry/internal/catalog/http"
	packagingapplication "skill_registry/internal/packaging/application"
	packaging "skill_registry/internal/packaging/domain"
	internalpackaginghttp "skill_registry/internal/packaging/http"
	platformaudit "skill_registry/internal/platform/audit"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
	platformaccess "skill_registry/internal/platform/http/access"
	persist "skill_registry/internal/platform/persistence"
	platformsession "skill_registry/internal/platform/session"
)

func contextWithRequestMetadataForTest(ctx context.Context, requestID string, traceID string) context.Context {
	ctx = platformhttp.ContextWithRequestMetadata(ctx, requestID, traceID)
	return ctx
}

type catalogReadModuleStub struct {
	listSkillsCalls atomic.Int64
}

func newCatalogReadModuleStubForTest(_ interface{ Helper() }) *catalogReadModuleStub {
	return &catalogReadModuleStub{}
}

func newTestServerWithCatalogReadModule(_ interface{ Helper() }, module internalcataloghttp.Workflow) *Server {
	server := newTestServer()
	server.catalogReadModule = module
	return server
}

func (m *catalogReadModuleStub) ListSkills(context.Context, auth.Principal, internalcataloghttp.SkillsQuery) (catalog.SkillsPage, *catalog.Error) {
	m.listSkillsCalls.Add(1)
	return catalog.SkillsPage{}, nil
}

func (m *catalogReadModuleStub) ListCuratedSkills(context.Context, auth.Principal, int, int) (catalog.CuratedSkillsPage, *catalog.Error) {
	return catalog.CuratedSkillsPage{}, nil
}

func (m *catalogReadModuleStub) GetSkill(context.Context, string) (catalog.SkillDetail, error) {
	return catalog.SkillDetail{}, nil
}

func (m *catalogReadModuleStub) ListVersions(context.Context, catalog.VersionQuery) (catalog.VersionsPage, error) {
	return catalog.VersionsPage{}, nil
}

func (m *catalogReadModuleStub) GetVersion(context.Context, string, string) (catalog.SkillVersionDetail, error) {
	return catalog.SkillVersionDetail{}, nil
}

func (m *catalogReadModuleStub) ResolveInstallTarget(context.Context, string, string) (catalog.InstallResolution, error) {
	return catalog.InstallResolution{}, nil
}

func (m *catalogReadModuleStub) listSkillsCallCount() int {
	return int(m.listSkillsCalls.Load())
}

type sessionModuleStub struct {
	currentCalls atomic.Int64
	loginCalls   atomic.Int64
	logoutCalls  atomic.Int64
}

func newSessionModuleStubForTest(_ interface{ Helper() }) *sessionModuleStub {
	return &sessionModuleStub{}
}

func newTestServerWithSessionModule(_ interface{ Helper() }, module platformsession.Workflow) *Server {
	server := newTestServer()
	server.sessionModule = module
	return server
}

func (m *sessionModuleStub) Current(auth.Principal) platformsession.Response {
	m.currentCalls.Add(1)
	return platformsession.Response{
		Role:            "publisher",
		Actor:           "delegated",
		AuthMechanism:   "delegated_session",
		AllowedActions:  []string{"delegated.action"},
		AllowedPackages: []string{},
	}
}

func (m *sessionModuleStub) Login(_ *http.Request, _, _ string) (auth.Principal, *http.Cookie, bool) {
	m.loginCalls.Add(1)
	return auth.Principal{
		Role:          "publisher",
		Actor:         "delegated",
		AuthMechanism: "delegated_session",
	}, &http.Cookie{Name: auth.SessionCookieName, Value: "delegated-cookie", Path: "/"}, true
}

func (m *sessionModuleStub) LogoutCookie(_ *http.Request) *http.Cookie {
	m.logoutCalls.Add(1)
	return &http.Cookie{Name: auth.SessionCookieName, Value: "", Path: "/", MaxAge: -1}
}

func (m *sessionModuleStub) Enabled() bool {
	return true
}

func (m *sessionModuleStub) currentCallCount() int {
	return int(m.currentCalls.Load())
}

func (m *sessionModuleStub) loginCallCount() int {
	return int(m.loginCalls.Load())
}

func (m *sessionModuleStub) logoutCallCount() int {
	return int(m.logoutCalls.Load())
}

type testPackagingWorkflow struct {
	*internalpackaginghttp.Workflow
	service *stubPackagingService
}

type stubPackagingService struct {
	prepareCalls atomic.Int64
	prepareFunc  func(context.Context, packaging.PrepareRequest) (packaging.PrepareResponse, error)
	validateFunc func(context.Context, packaging.ValidateRequest) (packaging.ValidateResponse, error)
	packFunc     func(context.Context, packaging.PackRequest) (packaging.PackResponse, error)
}

func (s *stubPackagingService) PrepareContext(ctx context.Context, input packaging.PrepareRequest) (packaging.PrepareResponse, error) {
	s.prepareCalls.Add(1)
	if s.prepareFunc != nil {
		return s.prepareFunc(ctx, input)
	}
	return packaging.PrepareResponse{}, nil
}

func (s *stubPackagingService) ValidateContext(ctx context.Context, input packaging.ValidateRequest) (packaging.ValidateResponse, error) {
	if s.validateFunc != nil {
		return s.validateFunc(ctx, input)
	}
	return packaging.ValidateResponse{}, nil
}

func (s *stubPackagingService) PackContext(ctx context.Context, input packaging.PackRequest) (packaging.PackResponse, error) {
	if s.packFunc != nil {
		return s.packFunc(ctx, input)
	}
	return packaging.PackResponse{}, nil
}

func newPackagingWorkflowForTest(_ interface{ Helper() }) *testPackagingWorkflow {
	return newPackagingWorkflowWithAuditForTest(nil)
}

func newPackagingWorkflowWithAuditForTest(auditRecorder *platformaudit.Recorder) *testPackagingWorkflow {
	service := &stubPackagingService{
		prepareFunc: func(_ context.Context, input packaging.PrepareRequest) (packaging.PrepareResponse, error) {
			return packaging.PrepareResponse{
				WorkspacePath: "cached-workspace",
				PackageName:   input.Target.PackageName,
				Version:       input.Target.Version,
				Warnings:      []string{},
				Errors:        []string{},
			}, nil
		},
	}
	return &testPackagingWorkflow{
		Workflow: internalpackaginghttp.NewWorkflow(internalpackaginghttp.WorkflowConfig{
			Service:             service,
			Audit:               auditRecorder,
			CacheTTL:            10 * time.Minute,
			AllowAnonymousCache: true,
		}),
		service: service,
	}
}

func (m *testPackagingWorkflow) prepareCallCount() int {
	return int(m.service.prepareCalls.Load())
}

func prepareManagedWorkspaceForTest(t *testing.T, service *packagingapplication.Service, packageName string, version string) (string, string) {
	t.Helper()

	fixtureDir := createPackagingFixture(t)
	prepareResponse, err := service.PrepareContext(context.Background(), packaging.PrepareRequest{
		Source: packaging.Source{
			Type:          "upload_archive",
			ArchiveBase64: buildSourceArchiveBase64FromDirectory(t, fixtureDir),
		},
		Target: packaging.Target{
			PackageName: packageName,
			Version:     version,
		},
		Metadata: packaging.Metadata{
			DisplayName: "Fixture Skill",
			Description: "Fixture packaging flow",
			SourceType:  "imported",
		},
	})
	if err != nil {
		t.Fatalf("prepare workspace: %v", err)
	}

	packResponse, err := service.PackContext(context.Background(), packaging.PackRequest{
		WorkspacePath: prepareResponse.WorkspacePath,
		Target: packaging.Target{
			PackageName: packageName,
			Version:     version,
		},
	})
	if err != nil {
		t.Fatalf("pack workspace: %v", err)
	}

	return prepareResponse.WorkspacePath, packResponse.BundlePath
}

type accessWorkflowStub struct {
	allow                 bool
	authorizeActionCalls  atomic.Int64
	authorizePackageCalls atomic.Int64
	denyCalls             atomic.Int64
}

func newAccessWorkflowStubForTest(_ interface{ Helper() }, allow bool) *accessWorkflowStub {
	return &accessWorkflowStub{allow: allow}
}

func (w *accessWorkflowStub) AuthorizeAction(writer http.ResponseWriter, request *http.Request, action auth.Action, packageName string, version string) bool {
	w.authorizeActionCalls.Add(1)
	if w.allow {
		return true
	}
	w.Deny(writer, request, auth.PrincipalFromContext(request.Context()), (&auth.Authorizer{}).AuthorizeAction(auth.Principal{Role: "reader"}, action), packageName, version)
	return false
}

func (w *accessWorkflowStub) AuthorizePackage(writer http.ResponseWriter, request *http.Request, packageName string) bool {
	w.authorizePackageCalls.Add(1)
	if w.allow {
		return true
	}
	w.Deny(writer, request, auth.PrincipalFromContext(request.Context()), (&auth.Authorizer{}).AuthorizePackage(auth.Principal{}, packageName), packageName, "")
	return false
}

func (w *accessWorkflowStub) Deny(writer http.ResponseWriter, request *http.Request, _ auth.Principal, authErr *auth.AuthorizationError, _ string, _ string) {
	w.denyCalls.Add(1)
	if authErr == nil {
		authErr = (&auth.AuthorizationError{
			Status:  http.StatusForbidden,
			Code:    "forbidden",
			Message: "forbidden",
		})
	}
	platformhttp.WriteAuthorizationError(writer, request, authErr)
}

func (w *accessWorkflowStub) denyCallCount() int {
	return int(w.denyCalls.Load())
}

var _ platformaccess.Gate = (*accessWorkflowStub)(nil)

type auditQueryModuleStub struct {
	listEventsCalls atomic.Int64
}

func newAuditQueryModuleStubForTest(_ interface{ Helper() }) *auditQueryModuleStub {
	return &auditQueryModuleStub{}
}

func (m *auditQueryModuleStub) ListEvents(context.Context, auth.Principal, url.Values) (persist.AuditPage, *catalog.Error) {
	m.listEventsCalls.Add(1)
	return persist.AuditPage{}, nil
}

func (m *auditQueryModuleStub) listEventsCallCount() int {
	return int(m.listEventsCalls.Load())
}

type catalogMutationWorkflowStub struct {
	createReleaseRecordCalls    atomic.Int64
	applySkillGovernanceCalls   atomic.Int64
	applyVersionGovernanceCalls atomic.Int64
}

func newCatalogMutationWorkflowStubForTest(_ interface{ Helper() }) *catalogMutationWorkflowStub {
	return &catalogMutationWorkflowStub{}
}

func (m *catalogMutationWorkflowStub) createReleaseRecord(context.Context, auth.Principal, string, string, []byte) (catalog.ReleaseRecordResult, int, *catalog.Error) {
	m.createReleaseRecordCalls.Add(1)
	return catalog.ReleaseRecordResult{Created: true}, http.StatusCreated, nil
}

func (m *catalogMutationWorkflowStub) applySkillGovernanceChange(context.Context, auth.Principal, string, []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
	m.applySkillGovernanceCalls.Add(1)
	return catalog.GovernanceChangeResponse{}, http.StatusCreated, nil
}

func (m *catalogMutationWorkflowStub) applyVersionGovernanceChange(context.Context, auth.Principal, string, string, []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
	m.applyVersionGovernanceCalls.Add(1)
	return catalog.GovernanceChangeResponse{}, http.StatusCreated, nil
}

func (m *catalogMutationWorkflowStub) createReleaseRecordCallCount() int {
	return int(m.createReleaseRecordCalls.Load())
}

func (m *catalogMutationWorkflowStub) applySkillGovernanceCallCount() int {
	return int(m.applySkillGovernanceCalls.Load())
}

func (m *catalogMutationWorkflowStub) applyVersionGovernanceCallCount() int {
	return int(m.applyVersionGovernanceCalls.Load())
}

type packagingAccessWorkflowStub struct {
	cleanupCalls        atomic.Int64
	downloadBundleCalls atomic.Int64
	bundlePath          string
}

func newPackagingAccessWorkflowStubForTest(t interface {
	Helper()
	TempDir() string
}) *packagingAccessWorkflowStub {
	t.Helper()
	bundlePath := filepath.Join(t.TempDir(), "bundle.tgz")
	if err := os.WriteFile(bundlePath, []byte("bundle"), 0o644); err != nil {
		panic(err)
	}
	return &packagingAccessWorkflowStub{bundlePath: bundlePath}
}

func (m *packagingAccessWorkflowStub) cleanup(context.Context, auth.Principal, packaging.CleanupRequest) (packaging.CleanupResponse, int, *catalog.Error) {
	m.cleanupCalls.Add(1)
	return packaging.CleanupResponse{Removed: true}, http.StatusOK, nil
}

func (m *packagingAccessWorkflowStub) downloadBundle(context.Context, auth.Principal, string, string, string) (string, *catalog.Error) {
	m.downloadBundleCalls.Add(1)
	return m.bundlePath, nil
}

func (m *packagingAccessWorkflowStub) downloadBundleCallCount() int {
	return int(m.downloadBundleCalls.Load())
}

func (m *packagingAccessWorkflowStub) cleanupCallCount() int {
	return int(m.cleanupCalls.Load())
}

type captureAuditStore struct {
	events []persist.AuditEvent
}

func (s *captureAuditStore) Append(_ context.Context, event persist.AuditEvent) error {
	s.events = append(s.events, event)
	return nil
}

func (s *captureAuditStore) List(context.Context, persist.AuditQuery) (persist.AuditPage, error) {
	return persist.AuditPage{}, nil
}
