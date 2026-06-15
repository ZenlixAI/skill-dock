package catalogapi

import (
	"context"
	"net/http"
	"strings"

	"skill_registry/app/catalogapi/common"
	"skill_registry/business/catalog"
	"skill_registry/business/packaging"
	internaldomain "skill_registry/internal/catalog/domain"
	packagingapplication "skill_registry/internal/packaging/application"
	"skill_registry/internal/platform/auth"
)

type catalogMutationService interface {
	CreateReleaseRecord(ctx context.Context, packageName string, version string, input internaldomain.ReleaseRecordInput) (internaldomain.ReleaseRecordResult, error)
	ApplySkillGovernanceChange(ctx context.Context, packageName string, input internaldomain.SkillGovernanceChangeInput, actor string) (internaldomain.GovernanceChangeResponse, error)
	ApplyVersionGovernanceChange(ctx context.Context, packageName string, version string, input internaldomain.VersionGovernanceChangeInput, actor string) (internaldomain.GovernanceChangeResponse, error)
}

type catalogGovernanceService interface {
	ListGovernanceChanges(ctx context.Context, query catalog.GovernanceQuery) (catalog.GovernancePage, error)
}

type packagingAccessService = packagingapplication.AccessService

func (s *Server) listGovernanceChanges(ctx context.Context, principal auth.Principal, query catalog.GovernanceQuery) (catalog.GovernancePage, *catalog.Error) {
	if authErr := s.authorizer.RequirePackageFilter(principal, query.PackageName); authErr != nil {
		return catalog.GovernancePage{}, authErr.CatalogError()
	}

	response, err := s.catalogGovernanceService.ListGovernanceChanges(ctx, query)
	if err != nil {
		return catalog.GovernancePage{}, common.CatalogErrorFromError(err)
	}
	response.Items = auth.FilterGovernanceRecordsByPrincipal(response.Items, principal)
	return response, nil
}

func (s *Server) createReleaseRecord(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error) {
	version, err := common.DecodeVersion(versionPath)
	if err != nil {
		return catalog.ReleaseRecordResult{}, 0, err
	}

	input, decodeErr := common.DecodeOptionalReleaseRecordInputBytes(rawBody)
	if decodeErr != nil {
		s.recordMutationAuditForEndpoint(ctx, principal, http.MethodPut, "/api/v1/skills/"+packageName+"/versions/"+version+"/release-record", "release.record", packageName, version, decodeErr.Status, "failed", map[string]any{
			"errorCode": decodeErr.Code,
		})
		return catalog.ReleaseRecordResult{}, 0, decodeErr
	}

	response, repoErr := s.catalogMutationService.CreateReleaseRecord(ctx, packageName, version, input)
	if repoErr != nil {
		catalogErr := common.CatalogErrorFromError(repoErr)
		s.recordMutationAuditForEndpoint(ctx, principal, http.MethodPut, "/api/v1/skills/"+packageName+"/versions/"+version+"/release-record", "release.record", packageName, version, catalogErr.Status, "failed", map[string]any{
			"errorCode": catalogErr.Code,
		})
		return catalog.ReleaseRecordResult{}, 0, catalogErr
	}

	status := http.StatusOK
	if response.Created {
		status = http.StatusCreated
	}
	s.recordMutationAuditForEndpoint(ctx, principal, http.MethodPut, "/api/v1/skills/"+packageName+"/versions/"+version+"/release-record", "release.record", packageName, version, status, "succeeded", map[string]any{
		"created":          response.Created,
		"idempotentReplay": response.IdempotentReplay,
	})
	return response, status, nil
}

func (s *Server) applySkillGovernanceChange(ctx context.Context, principal auth.Principal, packageName string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
	var payload catalog.SkillGovernanceChangeInput
	if err := common.DecodeJSONBytes(rawBody, &payload); err != nil {
		return catalog.GovernanceChangeResponse{}, 0, err
	}

	response, repoErr := s.catalogMutationService.ApplySkillGovernanceChange(ctx, packageName, payload, principal.Actor)
	if repoErr != nil {
		catalogErr := common.CatalogErrorFromError(repoErr)
		s.recordMutationAuditForEndpoint(ctx, principal, http.MethodPost, "/api/v1/skills/"+packageName+"/governance-changes", "governance.skill", packageName, "", catalogErr.Status, "failed", map[string]any{
			"action":    payload.Action,
			"errorCode": catalogErr.Code,
		})
		return catalog.GovernanceChangeResponse{}, 0, catalogErr
	}

	s.recordMutationAuditForEndpoint(ctx, principal, http.MethodPost, "/api/v1/skills/"+packageName+"/governance-changes", "governance.skill", packageName, "", http.StatusCreated, "succeeded", map[string]any{
		"action": payload.Action,
	})
	return response, http.StatusCreated, nil
}

func (s *Server) applyVersionGovernanceChange(ctx context.Context, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error) {
	version, err := common.DecodeVersion(versionPath)
	if err != nil {
		return catalog.GovernanceChangeResponse{}, 0, err
	}

	var payload catalog.VersionGovernanceChangeInput
	if err := common.DecodeJSONBytes(rawBody, &payload); err != nil {
		return catalog.GovernanceChangeResponse{}, 0, err
	}

	response, repoErr := s.catalogMutationService.ApplyVersionGovernanceChange(ctx, packageName, version, payload, principal.Actor)
	if repoErr != nil {
		catalogErr := common.CatalogErrorFromError(repoErr)
		s.recordMutationAuditForEndpoint(ctx, principal, http.MethodPost, "/api/v1/skills/"+packageName+"/versions/"+version+"/governance-changes", "governance.version", packageName, version, catalogErr.Status, "failed", map[string]any{
			"action":    payload.Action,
			"errorCode": catalogErr.Code,
		})
		return catalog.GovernanceChangeResponse{}, 0, catalogErr
	}

	s.recordMutationAuditForEndpoint(ctx, principal, http.MethodPost, "/api/v1/skills/"+packageName+"/versions/"+version+"/governance-changes", "governance.version", packageName, version, http.StatusCreated, "succeeded", map[string]any{
		"action": payload.Action,
	})
	return response, http.StatusCreated, nil
}

func (s *Server) cleanupPackaging(ctx context.Context, principal auth.Principal, payload packaging.CleanupRequest) (packaging.CleanupResponse, int, *catalog.Error) {
	if strings.TrimSpace(payload.WorkspacePath) == "" {
		err := catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "workspacePath is required.", nil)
		s.recordMutationAuditForEndpoint(ctx, principal, http.MethodPost, "/api/v1/skill-packaging/cleanup", "packaging.cleanup", "", "", err.Status, "failed", map[string]any{"errorCode": err.Code})
		return packaging.CleanupResponse{}, 0, err
	}
	if s.packaging == nil {
		return packaging.CleanupResponse{}, http.StatusNotFound, nil
	}
	if err := s.packaging.CleanupContext(ctx, payload.WorkspacePath); err != nil {
		catalogErr := common.CatalogErrorFromError(err)
		s.recordMutationAuditForEndpoint(ctx, principal, http.MethodPost, "/api/v1/skill-packaging/cleanup", "packaging.cleanup", "", "", catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
		return packaging.CleanupResponse{}, 0, catalogErr
	}
	s.recordMutationAuditForEndpoint(ctx, principal, http.MethodPost, "/api/v1/skill-packaging/cleanup", "packaging.cleanup", "", "", http.StatusOK, "succeeded", map[string]any{"workspacePath": payload.WorkspacePath})
	return packaging.CleanupResponse{WorkspacePath: payload.WorkspacePath, Removed: true}, http.StatusOK, nil
}

func (s *Server) downloadBundle(ctx context.Context, principal auth.Principal, workspacePath string, packageName string, version string) (string, *catalog.Error) {
	if strings.TrimSpace(workspacePath) == "" || strings.TrimSpace(packageName) == "" || strings.TrimSpace(version) == "" {
		err := catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "workspacePath, packageName, and version are required.", nil)
		s.recordMutationAuditForEndpoint(ctx, principal, http.MethodGet, "/api/v1/skill-packaging/bundle", "packaging.bundle.download", packageName, version, err.Status, "failed", map[string]any{"errorCode": err.Code})
		return "", err
	}
	if s.packaging == nil {
		return "", catalog.NewError(http.StatusNotFound, "not_found", "Not found.", nil)
	}

	bundlePath, err := s.packaging.BundleFilePathContext(ctx, workspacePath, packaging.Target{
		PackageName: packageName,
		Version:     version,
	})
	if err != nil {
		catalogErr := common.CatalogErrorFromError(err)
		s.recordMutationAuditForEndpoint(ctx, principal, http.MethodGet, "/api/v1/skill-packaging/bundle", "packaging.bundle.download", packageName, version, catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
		return "", catalogErr
	}

	s.recordMutationAuditForEndpoint(ctx, principal, http.MethodGet, "/api/v1/skill-packaging/bundle", "packaging.bundle.download", packageName, version, http.StatusOK, "succeeded", nil)
	return bundlePath, nil
}

func (s *Server) publicInstallTarget(response catalog.InstallResolution) catalog.InstallResolution {
	if s.publicNPMRegistryURL == "" {
		return response
	}

	internalRegistry := strings.TrimRight(strings.TrimSpace(response.NPMRegistry), "/")
	response.NPMRegistry = s.publicNPMRegistryURL
	response.TarballURL = rewriteRegistryBaseURL(response.TarballURL, internalRegistry, s.publicNPMRegistryURL)
	return response
}

func rewriteRegistryBaseURL(rawURL string, fromBase string, toBase string) string {
	trimmedURL := strings.TrimSpace(rawURL)
	normalizedFrom := strings.TrimRight(strings.TrimSpace(fromBase), "/")
	normalizedTo := strings.TrimRight(strings.TrimSpace(toBase), "/")

	switch {
	case trimmedURL == "":
		return rawURL
	case normalizedFrom == "":
		return trimmedURL
	case trimmedURL == normalizedFrom:
		return normalizedTo
	case strings.HasPrefix(trimmedURL, normalizedFrom+"/"):
		return normalizedTo + strings.TrimPrefix(trimmedURL, normalizedFrom)
	default:
		return trimmedURL
	}
}

func (s *Server) recordMutationAuditForEndpoint(ctx context.Context, principal auth.Principal, method string, endpoint string, action string, packageName string, version string, status int, outcome string, details map[string]any) {
	if s == nil || s.auditRecorder == nil {
		return
	}
	_ = s.auditRecorder.RecordMutationForEndpoint(ctx, principal, method, endpoint, action, packageName, version, status, outcome, details)
}
