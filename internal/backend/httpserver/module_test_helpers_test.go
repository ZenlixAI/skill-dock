package httpserver

import (
	"context"
	"net/http"
	"net/url"
	"sync/atomic"

	catalog "skill_registry/internal/catalog/domain"
	internalcataloghttp "skill_registry/internal/catalog/http"
	packagingapplication "skill_registry/internal/packaging/application"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
	platformaccess "skill_registry/internal/platform/http/access"
	persist "skill_registry/internal/platform/persistence"
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

func newTestServerWithCatalogReadModule(_ interface{ Helper() }, module *catalogReadModuleStub) *Server {
	server := newTestServer()
	server.listSkillsFunc = module.ListSkills
	server.listCuratedSkillsFunc = module.ListCuratedSkills
	server.getSkillFunc = module.GetSkill
	server.listVersionsFunc = module.ListVersions
	server.getVersionFunc = module.GetVersion
	server.resolveInstallTargetFunc = module.ResolveInstallTarget
	return server
}

func newImportReviewFuncForTest() ImportReviewFunc {
	return packagingapplication.NewImportReviewService(packagingapplication.NewWorkspaceService("", nil)).ReviewContext
}

func newPublishFuncForTest() PublishFunc {
	return packagingapplication.NewPublishService(packagingapplication.NewWorkspaceService("", nil), nil, nil).PublishContext
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

func (m *catalogMutationWorkflowStub) ListGovernanceChanges(context.Context, auth.Principal, catalog.GovernanceQuery) (catalog.GovernancePage, *catalog.Error) {
	return catalog.GovernancePage{}, nil
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
