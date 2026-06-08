package catalogapi

import (
	"context"
	"net/http"
	"strings"

	"skill_registry/app/auth"
	"skill_registry/business/catalog"
	"skill_registry/business/packaging"
)

type packagingAccessModule struct {
	service *packaging.Service
	audit   *AuditRecorder
}

type packagingAccessWorkflow interface {
	cleanup(ctx context.Context, principal auth.Principal, payload packaging.CleanupRequest) (packaging.CleanupResponse, int, *catalog.Error)
	downloadBundle(ctx context.Context, principal auth.Principal, workspacePath string, packageName string, version string) ([]byte, *catalog.Error)
}

func (m *packagingAccessModule) cleanup(ctx context.Context, principal auth.Principal, payload packaging.CleanupRequest) (packaging.CleanupResponse, int, *catalog.Error) {
	if strings.TrimSpace(payload.WorkspacePath) == "" {
		err := catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "workspacePath is required.", nil)
		m.recordAudit(ctx, principal, "packaging.cleanup", "", "", err.Status, "failed", map[string]any{"errorCode": err.Code})
		return packaging.CleanupResponse{}, 0, err
	}
	if err := m.service.Cleanup(payload.WorkspacePath); err != nil {
		catalogErr := catalogErrorFromError(err)
		m.recordAudit(ctx, principal, "packaging.cleanup", "", "", catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
		return packaging.CleanupResponse{}, 0, catalogErr
	}
	m.recordAudit(ctx, principal, "packaging.cleanup", "", "", http.StatusOK, "succeeded", map[string]any{"workspacePath": payload.WorkspacePath})
	return packaging.CleanupResponse{WorkspacePath: payload.WorkspacePath, Removed: true}, http.StatusOK, nil
}

func (m *packagingAccessModule) downloadBundle(ctx context.Context, principal auth.Principal, workspacePath string, packageName string, version string) ([]byte, *catalog.Error) {
	if strings.TrimSpace(workspacePath) == "" || strings.TrimSpace(packageName) == "" || strings.TrimSpace(version) == "" {
		err := catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "workspacePath, packageName, and version are required.", nil)
		m.recordAudit(ctx, principal, "packaging.bundle.download", packageName, version, err.Status, "failed", map[string]any{"errorCode": err.Code})
		return nil, err
	}

	content, err := m.service.BundleContent(workspacePath, packaging.Target{
		PackageName: packageName,
		Version:     version,
	})
	if err != nil {
		catalogErr := catalogErrorFromError(err)
		m.recordAudit(ctx, principal, "packaging.bundle.download", packageName, version, catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
		return nil, catalogErr
	}

	m.recordAudit(ctx, principal, "packaging.bundle.download", packageName, version, http.StatusOK, "succeeded", nil)
	return content, nil
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
