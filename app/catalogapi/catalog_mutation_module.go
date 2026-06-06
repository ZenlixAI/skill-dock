package catalogapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"skill_registry/app/auth"
	"skill_registry/business/catalog"
)

type catalogMutationModule struct {
	repo  catalog.WriteRepository
	audit *AuditRecorder
}

type catalogMutationWorkflow interface {
	createReleaseRecord(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error)
	applySkillGovernanceChange(ctx context.Context, principal auth.Principal, packageName string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error)
	applyVersionGovernanceChange(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error)
}

func (m *catalogMutationModule) createReleaseRecord(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error) {
	version, err := decodeVersion(versionPath)
	if err != nil {
		return catalog.ReleaseRecordResult{}, 0, err
	}

	input, decodeErr := decodeOptionalReleaseRecordInputBytes(rawBody)
	if decodeErr != nil {
		m.recordAudit(ctx, principal, http.MethodPut, "/api/v1/skills/"+packageName+"/versions/"+version+"/release-record", "release.record", packageName, version, decodeErr.Status, "failed", map[string]any{
			"errorCode": decodeErr.Code,
		})
		return catalog.ReleaseRecordResult{}, 0, decodeErr
	}

	response, repoErr := m.repo.CreateReleaseRecord(ctx, packageName, version, input)
	if repoErr != nil {
		catalogErr := catalogErrorFromError(repoErr)
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
	if err := decodeJSONBytes(rawBody, &payload); err != nil {
		return catalog.GovernanceChangeResponse{}, 0, err
	}

	response, repoErr := m.repo.ApplySkillGovernanceChange(ctx, packageName, payload, principal.Actor)
	if repoErr != nil {
		catalogErr := catalogErrorFromError(repoErr)
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
	version, err := decodeVersion(versionPath)
	if err != nil {
		return catalog.GovernanceChangeResponse{}, 0, err
	}

	var payload catalog.VersionGovernanceChangeInput
	if err := decodeJSONBytes(rawBody, &payload); err != nil {
		return catalog.GovernanceChangeResponse{}, 0, err
	}

	response, repoErr := m.repo.ApplyVersionGovernanceChange(ctx, packageName, version, payload, principal.Actor)
	if repoErr != nil {
		catalogErr := catalogErrorFromError(repoErr)
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

func decodeOptionalReleaseRecordInputBytes(rawBody []byte) (catalog.ReleaseRecordInput, *catalog.Error) {
	if len(bytes.TrimSpace(rawBody)) == 0 {
		return catalog.ReleaseRecordInput{}, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(rawBody))
	decoder.DisallowUnknownFields()

	var payload catalog.ReleaseRecordInput
	if err := decoder.Decode(&payload); err != nil {
		if err == io.EOF {
			return catalog.ReleaseRecordInput{}, nil
		}
		return catalog.ReleaseRecordInput{}, catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "request body is invalid JSON.", map[string]any{"error": err.Error()})
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return catalog.ReleaseRecordInput{}, catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "request body must contain a single JSON object.", nil)
	}
	return payload, nil
}
