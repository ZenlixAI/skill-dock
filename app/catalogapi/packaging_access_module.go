package catalogapi

import (
	"context"
	"net/http"
	"strings"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/audit"
	"skill_registry/app/catalogapi/common"
	"skill_registry/business/catalog"
	"skill_registry/business/packaging"
)

type packagingAccessModule struct {
	service packagingAccessService
	audit   *audit.Recorder
}

type packagingAccessService interface {
	CleanupContext(ctx context.Context, workspacePath string) error
	BundleFilePathContext(ctx context.Context, workspacePath string, target packaging.Target) (string, error)
}

type packagingAccessWorkflow interface {
	cleanup(ctx context.Context, principal auth.Principal, payload packaging.CleanupRequest) (packaging.CleanupResponse, int, *catalog.Error)
	downloadBundle(ctx context.Context, principal auth.Principal, workspacePath string, packageName string, version string) (string, *catalog.Error)
}

func (m *packagingAccessModule) cleanup(ctx context.Context, principal auth.Principal, payload packaging.CleanupRequest) (packaging.CleanupResponse, int, *catalog.Error) {
	if strings.TrimSpace(payload.WorkspacePath) == "" {
		err := catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "workspacePath is required.", nil)
		m.recordAudit(ctx, principal, "packaging.cleanup", "", "", err.Status, "failed", map[string]any{"errorCode": err.Code})
		return packaging.CleanupResponse{}, 0, err
	}
	if err := m.service.CleanupContext(ctx, payload.WorkspacePath); err != nil {
		catalogErr := common.CatalogErrorFromError(err)
		m.recordAudit(ctx, principal, "packaging.cleanup", "", "", catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
		return packaging.CleanupResponse{}, 0, catalogErr
	}
	m.recordAudit(ctx, principal, "packaging.cleanup", "", "", http.StatusOK, "succeeded", map[string]any{"workspacePath": payload.WorkspacePath})
	return packaging.CleanupResponse{WorkspacePath: payload.WorkspacePath, Removed: true}, http.StatusOK, nil
}

func (m *packagingAccessModule) downloadBundle(ctx context.Context, principal auth.Principal, workspacePath string, packageName string, version string) (string, *catalog.Error) {
	if strings.TrimSpace(workspacePath) == "" || strings.TrimSpace(packageName) == "" || strings.TrimSpace(version) == "" {
		err := catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "workspacePath, packageName, and version are required.", nil)
		m.recordAudit(ctx, principal, "packaging.bundle.download", packageName, version, err.Status, "failed", map[string]any{"errorCode": err.Code})
		return "", err
	}

	bundlePath, err := m.service.BundleFilePathContext(ctx, workspacePath, packaging.Target{
		PackageName: packageName,
		Version:     version,
	})
	if err != nil {
		catalogErr := common.CatalogErrorFromError(err)
		m.recordAudit(ctx, principal, "packaging.bundle.download", packageName, version, catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
		return "", catalogErr
	}

	m.recordAudit(ctx, principal, "packaging.bundle.download", packageName, version, http.StatusOK, "succeeded", nil)
	return bundlePath, nil
}

func (m *packagingAccessModule) recordAudit(ctx context.Context, principal auth.Principal, action string, packageName string, version string, status int, outcome string, details map[string]any) {
	if m == nil || m.audit == nil {
		return
	}
	endpoint := "/api/v1/skill-packaging/cleanup"
	method := http.MethodPost
	if action == "packaging.bundle.download" {
		endpoint = "/api/v1/skill-packaging/bundle"
		method = http.MethodGet
	}
	_ = m.audit.RecordMutationForEndpoint(ctx, principal, method, endpoint, action, packageName, version, status, outcome, details)
}
