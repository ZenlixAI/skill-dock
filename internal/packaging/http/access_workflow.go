package http

import (
	"context"
	stdhttp "net/http"
	"strings"

	catalog "skill_registry/internal/catalog/domain"
	packagingapplication "skill_registry/internal/packaging/application"
	packaging "skill_registry/internal/packaging/domain"
	platformaudit "skill_registry/internal/platform/audit"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
)

type AccessService = packagingapplication.AccessService

type AccessWorkflow interface {
	Cleanup(ctx context.Context, principal auth.Principal, payload packaging.CleanupRequest) (packaging.CleanupResponse, int, *catalog.Error)
	DownloadBundle(ctx context.Context, principal auth.Principal, workspacePath string, packageName string, version string) (string, *catalog.Error)
}

type AccessModule struct {
	service AccessService
	audit   *platformaudit.Recorder
}

func NewAccessWorkflow(service AccessService, audit *platformaudit.Recorder) *AccessModule {
	return &AccessModule{
		service: service,
		audit:   audit,
	}
}

func (m *AccessModule) Cleanup(ctx context.Context, principal auth.Principal, payload packaging.CleanupRequest) (packaging.CleanupResponse, int, *catalog.Error) {
	if strings.TrimSpace(payload.WorkspacePath) == "" {
		err := catalog.NewError(stdhttp.StatusUnprocessableEntity, "invalid_request", "workspacePath is required.", nil)
		m.recordMutationAuditForEndpoint(ctx, principal, stdhttp.MethodPost, "/api/v1/skill-packaging/cleanup", "packaging.cleanup", "", "", err.Status, "failed", map[string]any{"errorCode": err.Code})
		return packaging.CleanupResponse{}, 0, err
	}
	if m.service == nil {
		return packaging.CleanupResponse{}, stdhttp.StatusNotFound, nil
	}
	if err := m.service.CleanupContext(ctx, payload.WorkspacePath); err != nil {
		catalogErr := platformhttp.CatalogErrorFromError(err)
		m.recordMutationAuditForEndpoint(ctx, principal, stdhttp.MethodPost, "/api/v1/skill-packaging/cleanup", "packaging.cleanup", "", "", catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
		return packaging.CleanupResponse{}, 0, catalogErr
	}
	m.recordMutationAuditForEndpoint(ctx, principal, stdhttp.MethodPost, "/api/v1/skill-packaging/cleanup", "packaging.cleanup", "", "", stdhttp.StatusOK, "succeeded", map[string]any{"workspacePath": payload.WorkspacePath})
	return packaging.CleanupResponse{WorkspacePath: payload.WorkspacePath, Removed: true}, stdhttp.StatusOK, nil
}

func (m *AccessModule) DownloadBundle(ctx context.Context, principal auth.Principal, workspacePath string, packageName string, version string) (string, *catalog.Error) {
	if strings.TrimSpace(workspacePath) == "" || strings.TrimSpace(packageName) == "" || strings.TrimSpace(version) == "" {
		err := catalog.NewError(stdhttp.StatusUnprocessableEntity, "invalid_request", "workspacePath, packageName, and version are required.", nil)
		m.recordMutationAuditForEndpoint(ctx, principal, stdhttp.MethodGet, "/api/v1/skill-packaging/bundle", "packaging.bundle.download", packageName, version, err.Status, "failed", map[string]any{"errorCode": err.Code})
		return "", err
	}
	if m.service == nil {
		return "", catalog.NewError(stdhttp.StatusNotFound, "not_found", "Not found.", nil)
	}

	bundlePath, err := m.service.BundleFilePathContext(ctx, workspacePath, packaging.Target{
		PackageName: packageName,
		Version:     version,
	})
	if err != nil {
		catalogErr := platformhttp.CatalogErrorFromError(err)
		m.recordMutationAuditForEndpoint(ctx, principal, stdhttp.MethodGet, "/api/v1/skill-packaging/bundle", "packaging.bundle.download", packageName, version, catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
		return "", catalogErr
	}

	m.recordMutationAuditForEndpoint(ctx, principal, stdhttp.MethodGet, "/api/v1/skill-packaging/bundle", "packaging.bundle.download", packageName, version, stdhttp.StatusOK, "succeeded", nil)
	return bundlePath, nil
}

func (m *AccessModule) recordMutationAuditForEndpoint(ctx context.Context, principal auth.Principal, method string, endpoint string, action string, packageName string, version string, status int, outcome string, details map[string]any) {
	if m == nil || m.audit == nil {
		return
	}
	_ = m.audit.RecordMutationForEndpoint(ctx, principal, method, endpoint, action, packageName, version, status, outcome, details)
}
