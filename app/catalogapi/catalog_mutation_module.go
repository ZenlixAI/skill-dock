package catalogapi

import (
	"context"
	"net/http"

	"skill_registry/app/catalogapi/common"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/audit"
	"skill_registry/business/catalog"
	internaldomain "skill_registry/internal/catalog/domain"
)

type catalogMutationService interface {
	CreateReleaseRecord(ctx context.Context, packageName string, version string, input internaldomain.ReleaseRecordInput) (internaldomain.ReleaseRecordResult, error)
	ApplySkillGovernanceChange(ctx context.Context, packageName string, input internaldomain.SkillGovernanceChangeInput, actor string) (internaldomain.GovernanceChangeResponse, error)
	ApplyVersionGovernanceChange(ctx context.Context, packageName string, version string, input internaldomain.VersionGovernanceChangeInput, actor string) (internaldomain.GovernanceChangeResponse, error)
}

type catalogMutationModule struct {
	repo    catalog.WriteRepository
	service catalogMutationService
	audit   *audit.Recorder
}

type catalogMutationWorkflow interface {
	createReleaseRecord(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error)
	applySkillGovernanceChange(ctx context.Context, principal auth.Principal, packageName string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error)
	applyVersionGovernanceChange(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error)
}

func (m *catalogMutationModule) createReleaseRecord(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error) {
	version, err := common.DecodeVersion(versionPath)
	if err != nil {
		return catalog.ReleaseRecordResult{}, 0, err
	}

	input, decodeErr := common.DecodeOptionalReleaseRecordInputBytes(rawBody)
	if decodeErr != nil {
		m.recordAudit(ctx, principal, http.MethodPut, "/api/v1/skills/"+packageName+"/versions/"+version+"/release-record", "release.record", packageName, version, decodeErr.Status, "failed", map[string]any{
			"errorCode": decodeErr.Code,
		})
		return catalog.ReleaseRecordResult{}, 0, decodeErr
	}

	response, repoErr := m.writeService().CreateReleaseRecord(ctx, packageName, version, input)
	if repoErr != nil {
		catalogErr := common.CatalogErrorFromError(repoErr)
		m.recordAudit(ctx, principal, http.MethodPut, "/api/v1/skills/"+packageName+"/versions/"+version+"/release-record", "release.record", packageName, version, catalogErr.Status, "failed", map[string]any{
			"errorCode": catalogErr.Code,
		})
		return catalog.ReleaseRecordResult{}, 0, catalogErr
	}

	status := http.StatusOK
	if response.Created {
		status = http.StatusCreated
	}
	m.recordAudit(ctx, principal, http.MethodPut, "/api/v1/skills/"+packageName+"/versions/"+version+"/release-record", "release.record", packageName, version, status, "succeeded", map[string]any{
		"created":          response.Created,
		"idempotentReplay": response.IdempotentReplay,
	})
	return response, status, nil
}

func (m *catalogMutationModule) applySkillGovernanceChange(ctx context.Context, principal auth.Principal, packageName string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
	var payload catalog.SkillGovernanceChangeInput
	if err := common.DecodeJSONBytes(rawBody, &payload); err != nil {
		return catalog.GovernanceChangeResponse{}, 0, err
	}

	response, repoErr := m.writeService().ApplySkillGovernanceChange(ctx, packageName, payload, principal.Actor)
	if repoErr != nil {
		catalogErr := common.CatalogErrorFromError(repoErr)
		m.recordAudit(ctx, principal, http.MethodPost, "/api/v1/skills/"+packageName+"/governance-changes", "governance.skill", packageName, "", catalogErr.Status, "failed", map[string]any{
			"action":    payload.Action,
			"errorCode": catalogErr.Code,
		})
		return catalog.GovernanceChangeResponse{}, 0, catalogErr
	}

	m.recordAudit(ctx, principal, http.MethodPost, "/api/v1/skills/"+packageName+"/governance-changes", "governance.skill", packageName, "", http.StatusCreated, "succeeded", map[string]any{
		"action": payload.Action,
	})
	return response, http.StatusCreated, nil
}

func (m *catalogMutationModule) applyVersionGovernanceChange(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
	version, err := common.DecodeVersion(versionPath)
	if err != nil {
		return catalog.GovernanceChangeResponse{}, 0, err
	}

	var payload catalog.VersionGovernanceChangeInput
	if err := common.DecodeJSONBytes(rawBody, &payload); err != nil {
		return catalog.GovernanceChangeResponse{}, 0, err
	}

	response, repoErr := m.writeService().ApplyVersionGovernanceChange(ctx, packageName, version, payload, principal.Actor)
	if repoErr != nil {
		catalogErr := common.CatalogErrorFromError(repoErr)
		m.recordAudit(ctx, principal, http.MethodPost, "/api/v1/skills/"+packageName+"/versions/"+version+"/governance-changes", "governance.version", packageName, version, catalogErr.Status, "failed", map[string]any{
			"action":    payload.Action,
			"errorCode": catalogErr.Code,
		})
		return catalog.GovernanceChangeResponse{}, 0, catalogErr
	}

	m.recordAudit(ctx, principal, http.MethodPost, "/api/v1/skills/"+packageName+"/versions/"+version+"/governance-changes", "governance.version", packageName, version, http.StatusCreated, "succeeded", map[string]any{
		"action": payload.Action,
	})
	return response, http.StatusCreated, nil
}

func (m *catalogMutationModule) recordAudit(ctx context.Context, principal auth.Principal, method string, endpoint string, action string, packageName string, version string, status int, outcome string, details map[string]any) {
	if m == nil || m.audit == nil {
		return
	}
	_ = m.audit.RecordMutationForEndpoint(ctx, principal, method, endpoint, action, packageName, version, status, outcome, details)
}

func (m *catalogMutationModule) writeService() catalogMutationService {
	if m != nil && m.service != nil {
		return m.service
	}
	return m.repo
}
