package http

import (
	"context"
	stdhttp "net/http"

	catalog "skill_registry/internal/catalog/domain"
	cataloghttp "skill_registry/internal/catalog/http"
	platformaudit "skill_registry/internal/platform/audit"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
)

type MutationService interface {
	CreateReleaseRecord(ctx context.Context, packageName string, version string, input catalog.ReleaseRecordInput) (catalog.ReleaseRecordResult, error)
	ApplySkillGovernanceChange(ctx context.Context, packageName string, input catalog.SkillGovernanceChangeInput, actor string) (catalog.GovernanceChangeResponse, error)
	ApplyVersionGovernanceChange(ctx context.Context, packageName string, version string, input catalog.VersionGovernanceChangeInput, actor string) (catalog.GovernanceChangeResponse, error)
}

type QueryService interface {
	ListGovernanceChanges(ctx context.Context, query catalog.GovernanceQuery) (catalog.GovernancePage, error)
}

type Workflow interface {
	ListGovernanceChanges(ctx context.Context, principal auth.Principal, query catalog.GovernanceQuery) (catalog.GovernancePage, *catalog.Error)
	CreateReleaseRecord(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error)
	ApplySkillGovernanceChange(ctx context.Context, principal auth.Principal, packageName string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error)
	ApplyVersionGovernanceChange(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error)
}

type Module struct {
	authorizer      *auth.Authorizer
	mutationService MutationService
	queryService    QueryService
	audit           *platformaudit.Recorder
}

func NewWorkflow(authorizer *auth.Authorizer, mutationService MutationService, queryService QueryService, audit *platformaudit.Recorder) *Module {
	return &Module{
		authorizer:      authorizer,
		mutationService: mutationService,
		queryService:    queryService,
		audit:           audit,
	}
}

func (m *Module) ListGovernanceChanges(ctx context.Context, principal auth.Principal, query catalog.GovernanceQuery) (catalog.GovernancePage, *catalog.Error) {
	if authErr := m.authorizer.RequirePackageFilter(principal, query.PackageName); authErr != nil {
		return catalog.GovernancePage{}, authErr.CatalogError()
	}

	response, err := m.queryService.ListGovernanceChanges(ctx, query)
	if err != nil {
		return catalog.GovernancePage{}, platformhttp.CatalogErrorFromError(err)
	}
	response.Items = auth.FilterGovernanceRecordsByPrincipal(response.Items, principal)
	return response, nil
}

func (m *Module) CreateReleaseRecord(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error) {
	version, err := cataloghttp.DecodeVersion(versionPath)
	if err != nil {
		return catalog.ReleaseRecordResult{}, 0, err
	}

	input, decodeErr := platformhttp.DecodeOptionalReleaseRecordInputBytes(rawBody)
	if decodeErr != nil {
		m.recordMutationAuditForEndpoint(ctx, principal, stdhttp.MethodPut, "/api/v1/skills/"+packageName+"/versions/"+version+"/release-record", "release.record", packageName, version, decodeErr.Status, "failed", map[string]any{
			"errorCode": decodeErr.Code,
		})
		return catalog.ReleaseRecordResult{}, 0, decodeErr
	}

	response, repoErr := m.mutationService.CreateReleaseRecord(ctx, packageName, version, input)
	if repoErr != nil {
		catalogErr := platformhttp.CatalogErrorFromError(repoErr)
		m.recordMutationAuditForEndpoint(ctx, principal, stdhttp.MethodPut, "/api/v1/skills/"+packageName+"/versions/"+version+"/release-record", "release.record", packageName, version, catalogErr.Status, "failed", map[string]any{
			"errorCode": catalogErr.Code,
		})
		return catalog.ReleaseRecordResult{}, 0, catalogErr
	}

	status := stdhttp.StatusOK
	if response.Created {
		status = stdhttp.StatusCreated
	}
	m.recordMutationAuditForEndpoint(ctx, principal, stdhttp.MethodPut, "/api/v1/skills/"+packageName+"/versions/"+version+"/release-record", "release.record", packageName, version, status, "succeeded", map[string]any{
		"created":          response.Created,
		"idempotentReplay": response.IdempotentReplay,
	})
	return response, status, nil
}

func (m *Module) ApplySkillGovernanceChange(ctx context.Context, principal auth.Principal, packageName string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
	var payload catalog.SkillGovernanceChangeInput
	if err := platformhttp.DecodeJSONBytes(rawBody, &payload); err != nil {
		return catalog.GovernanceChangeResponse{}, 0, err
	}

	response, repoErr := m.mutationService.ApplySkillGovernanceChange(ctx, packageName, payload, principal.Actor)
	if repoErr != nil {
		catalogErr := platformhttp.CatalogErrorFromError(repoErr)
		m.recordMutationAuditForEndpoint(ctx, principal, stdhttp.MethodPost, "/api/v1/skills/"+packageName+"/governance-changes", "governance.skill", packageName, "", catalogErr.Status, "failed", map[string]any{
			"action":    payload.Action,
			"errorCode": catalogErr.Code,
		})
		return catalog.GovernanceChangeResponse{}, 0, catalogErr
	}

	m.recordMutationAuditForEndpoint(ctx, principal, stdhttp.MethodPost, "/api/v1/skills/"+packageName+"/governance-changes", "governance.skill", packageName, "", stdhttp.StatusCreated, "succeeded", map[string]any{
		"action": payload.Action,
	})
	return response, stdhttp.StatusCreated, nil
}

func (m *Module) ApplyVersionGovernanceChange(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
	version, err := cataloghttp.DecodeVersion(versionPath)
	if err != nil {
		return catalog.GovernanceChangeResponse{}, 0, err
	}

	var payload catalog.VersionGovernanceChangeInput
	if err := platformhttp.DecodeJSONBytes(rawBody, &payload); err != nil {
		return catalog.GovernanceChangeResponse{}, 0, err
	}

	response, repoErr := m.mutationService.ApplyVersionGovernanceChange(ctx, packageName, version, payload, principal.Actor)
	if repoErr != nil {
		catalogErr := platformhttp.CatalogErrorFromError(repoErr)
		m.recordMutationAuditForEndpoint(ctx, principal, stdhttp.MethodPost, "/api/v1/skills/"+packageName+"/versions/"+version+"/governance-changes", "governance.version", packageName, version, catalogErr.Status, "failed", map[string]any{
			"action":    payload.Action,
			"errorCode": catalogErr.Code,
		})
		return catalog.GovernanceChangeResponse{}, 0, catalogErr
	}

	m.recordMutationAuditForEndpoint(ctx, principal, stdhttp.MethodPost, "/api/v1/skills/"+packageName+"/versions/"+version+"/governance-changes", "governance.version", packageName, version, stdhttp.StatusCreated, "succeeded", map[string]any{
		"action": payload.Action,
	})
	return response, stdhttp.StatusCreated, nil
}

func (m *Module) recordMutationAuditForEndpoint(ctx context.Context, principal auth.Principal, method string, endpoint string, action string, packageName string, version string, status int, outcome string, details map[string]any) {
	if m == nil || m.audit == nil {
		return
	}
	_ = m.audit.RecordMutationForEndpoint(ctx, principal, method, endpoint, action, packageName, version, status, outcome, details)
}
