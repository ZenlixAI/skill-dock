package catalogapi

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"skill_registry/app/catalogapi/audit"
	catalog "skill_registry/internal/catalog/domain"
	packagingapplication "skill_registry/internal/packaging/application"
	packaging "skill_registry/internal/packaging/domain"
	"skill_registry/internal/platform/auth"
)

func TestPackagingAccessModuleUsesPackagingApplicationAccessContract(t *testing.T) {
	var _ packagingapplication.AccessService = (*packagingapplication.Service)(nil)
}

type bundlePathOnlyPackagingService struct {
	bundlePath string
}

func (s *bundlePathOnlyPackagingService) PrepareContext(context.Context, packaging.PrepareRequest) (packaging.PrepareResponse, error) {
	return packaging.PrepareResponse{}, nil
}

func (s *bundlePathOnlyPackagingService) ValidateContext(context.Context, packaging.ValidateRequest) (packaging.ValidateResponse, error) {
	return packaging.ValidateResponse{}, nil
}

func (s *bundlePathOnlyPackagingService) PackContext(context.Context, packaging.PackRequest) (packaging.PackResponse, error) {
	return packaging.PackResponse{}, nil
}

func (s *bundlePathOnlyPackagingService) CleanupContext(context.Context, string) error {
	return nil
}

func (s *bundlePathOnlyPackagingService) BundleFilePathContext(context.Context, string, packaging.Target) (string, error) {
	return s.bundlePath, nil
}

func (s *bundlePathOnlyPackagingService) PublishContext(context.Context, packaging.PublishRequest) (packaging.PublishResponse, error) {
	return packaging.PublishResponse{}, nil
}

func (s *bundlePathOnlyPackagingService) ConfigurePublishWorkflow(packagingapplication.ReleaseRecordCreator, packagingapplication.PackagePublisher) {
}

func TestServerResolvesBundlePathAndRecordsAudit(t *testing.T) {
	tempDir := t.TempDir()
	bundlePath := filepath.Join(tempDir, "fixture.tgz")
	if err := os.WriteFile(bundlePath, []byte("bundle"), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	store := &captureAuditStore{}
	server := &Server{
		packaging:     &bundlePathOnlyPackagingService{bundlePath: bundlePath},
		auditRecorder: audit.NewRecorder(store),
	}
	principal := auth.Principal{
		Role:          "publisher",
		Actor:         "svc-publisher",
		AuthMechanism: "configured_token",
	}
	ctx := contextWithRequestMetadataForTest(context.Background(), "req_bundle", "trace_bundle")

	resolvedPath, err := server.downloadBundle(ctx, principal, "ws_fixture", "@platform.skill/fixture-skill", "0.2.0")
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

func TestServerReturnsBundleLoadError(t *testing.T) {
	server := &Server{
		packaging: &bundlePathOnlyPackagingService{bundlePath: ""},
	}

	_, err := server.downloadBundle(context.Background(), auth.Principal{}, "", "@platform.skill/fixture-skill", "0.2.0")
	if err == nil {
		t.Fatal("expected invalid request error")
	}
	if err.Code != "invalid_request" && err.Code != catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "", nil).Code {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func TestServerCleanupRemovesWorkspaceAndBundle(t *testing.T) {
	service := packagingapplication.NewService(t.TempDir())
	workspacePath, bundlePath := prepareManagedWorkspaceForTest(t, service, "@platform.skill/fixture-skill", "0.2.0")
	store := &captureAuditStore{}
	server := &Server{
		packaging:     service,
		auditRecorder: audit.NewRecorder(store),
	}
	principal := auth.Principal{
		Role:          "publisher",
		Actor:         "svc-publisher",
		AuthMechanism: "configured_token",
	}
	ctx := contextWithRequestMetadataForTest(context.Background(), "req_cleanup", "trace_cleanup")

	response, status, err := server.cleanupPackaging(ctx, principal, packaging.CleanupRequest{WorkspacePath: workspacePath})
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
