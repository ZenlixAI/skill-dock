package catalogapi

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"skill_registry/app/auth"
	"skill_registry/business/catalog"
	"skill_registry/business/packaging"
)

type bundlePathOnlyPackagingService struct {
	bundlePath string
}

func (s *bundlePathOnlyPackagingService) CleanupContext(context.Context, string) error {
	return nil
}

func (s *bundlePathOnlyPackagingService) BundleFilePathContext(context.Context, string, packaging.Target) (string, error) {
	return s.bundlePath, nil
}

func TestPackagingAccessModuleResolvesBundlePathAndRecordsAudit(t *testing.T) {
	tempDir := t.TempDir()
	bundlePath := filepath.Join(tempDir, "fixture.tgz")
	if err := os.WriteFile(bundlePath, []byte("bundle"), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	store := &captureAuditStore{}
	module := &packagingAccessModule{
		service: &bundlePathOnlyPackagingService{bundlePath: bundlePath},
		audit:   NewAuditRecorder(store),
	}
	principal := auth.Principal{
		Role:          "publisher",
		Actor:         "svc-publisher",
		AuthMechanism: "configured_token",
	}
	ctx := contextWithRequestMetadataForTest(context.Background(), "req_bundle", "trace_bundle")

	resolvedPath, err := module.downloadBundle(ctx, principal, "ws_fixture", "@platform.skill/fixture-skill", "0.2.0")

	if err != nil {
		t.Fatalf("download bundle: %v", err)
	}
	if resolvedPath != bundlePath {
		t.Fatalf("expected resolved bundle path %q, got %q", bundlePath, resolvedPath)
	}
	if len(store.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(store.events))
	}
	if store.events[0].Action != "packaging.bundle.download" || store.events[0].StatusCode != http.StatusOK {
		t.Fatalf("unexpected audit event: %+v", store.events[0])
	}
}

func TestPackagingAccessModuleReturnsBundleLoadError(t *testing.T) {
	module := &packagingAccessModule{
		service: &bundlePathOnlyPackagingService{bundlePath: ""},
	}

	_, err := module.downloadBundle(context.Background(), auth.Principal{}, "", "@platform.skill/fixture-skill", "0.2.0")
	if err == nil {
		t.Fatal("expected invalid request error")
	}
	if err.Code != "invalid_request" && err.Code != catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "", nil).Code {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func TestPackagingAccessModuleCleanupRemovesWorkspaceAndBundle(t *testing.T) {
	service := packaging.NewService(t.TempDir())
	workspacePath, bundlePath := prepareManagedWorkspaceForTest(t, service, "@platform.skill/fixture-skill", "0.2.0")
	store := &captureAuditStore{}
	module := &packagingAccessModule{
		service: service,
		audit:   NewAuditRecorder(store),
	}
	principal := auth.Principal{
		Role:          "publisher",
		Actor:         "svc-publisher",
		AuthMechanism: "configured_token",
	}
	ctx := contextWithRequestMetadataForTest(context.Background(), "req_cleanup", "trace_cleanup")

	response, status, err := module.cleanup(ctx, principal, packaging.CleanupRequest{WorkspacePath: workspacePath})

	if err != nil {
		t.Fatalf("cleanup workspace: %v", err)
	}
	if status != http.StatusOK || !response.Removed {
		t.Fatalf("unexpected cleanup response: status=%d response=%+v", status, response)
	}
	if _, err := os.Stat(workspacePath); !os.IsNotExist(err) {
		t.Fatalf("expected workspace removed, got err=%v", err)
	}
	if _, err := os.Stat(bundlePath); !os.IsNotExist(err) {
		t.Fatalf("expected bundle removed, got err=%v", err)
	}
	if len(store.events) != 1 || store.events[0].Action != "packaging.cleanup" {
		t.Fatalf("unexpected audit events: %+v", store.events)
	}
}
