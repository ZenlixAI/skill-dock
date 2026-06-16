package http

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	catalog "skill_registry/internal/catalog/domain"
	packagingapplication "skill_registry/internal/packaging/application"
	packaging "skill_registry/internal/packaging/domain"
	platformaudit "skill_registry/internal/platform/audit"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
	persist "skill_registry/internal/platform/persistence"
)

func TestAccessWorkflowUsesPackagingApplicationAccessContract(t *testing.T) {
	var _ packagingapplication.AccessService = (*packagingapplication.Service)(nil)
}

func TestAccessWorkflowResolvesBundlePathAndRecordsAudit(t *testing.T) {
	tempDir := t.TempDir()
	bundlePath := filepath.Join(tempDir, "fixture.tgz")
	if err := os.WriteFile(bundlePath, []byte("bundle"), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	store := &captureAuditStore{}
	workflow := NewAccessWorkflow(&bundlePathOnlyPackagingService{bundlePath: bundlePath}, platformaudit.NewRecorder(store))
	principal := auth.Principal{
		Role:          "publisher",
		Actor:         "svc-publisher",
		AuthMechanism: "configured_token",
	}
	ctx := contextWithRequestMetadataForTest(context.Background(), "req_bundle", "trace_bundle")

	resolvedPath, err := workflow.DownloadBundle(ctx, principal, "ws_fixture", "@platform.skill/fixture-skill", "0.2.0")
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

func TestAccessWorkflowReturnsBundleLoadError(t *testing.T) {
	workflow := NewAccessWorkflow(&bundlePathOnlyPackagingService{bundlePath: ""}, nil)

	_, err := workflow.DownloadBundle(context.Background(), auth.Principal{}, "", "@platform.skill/fixture-skill", "0.2.0")
	if err == nil {
		t.Fatal("expected invalid request error")
	}
	if err.Code != "invalid_request" && err.Code != catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "", nil).Code {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func TestAccessWorkflowCleanupRemovesWorkspaceAndBundle(t *testing.T) {
	workspacePath := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	bundlePath := filepath.Join(t.TempDir(), "bundle.tgz")
	if err := os.WriteFile(bundlePath, []byte("bundle"), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	store := &captureAuditStore{}
	workflow := NewAccessWorkflow(&cleanupOnlyPackagingService{
		cleanupFunc: func(path string) error {
			if err := os.RemoveAll(path); err != nil {
				return err
			}
			return os.Remove(bundlePath)
		},
	}, platformaudit.NewRecorder(store))
	principal := auth.Principal{
		Role:          "publisher",
		Actor:         "svc-publisher",
		AuthMechanism: "configured_token",
	}
	ctx := contextWithRequestMetadataForTest(context.Background(), "req_cleanup", "trace_cleanup")

	response, status, err := workflow.Cleanup(ctx, principal, packaging.CleanupRequest{WorkspacePath: workspacePath})
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

type cleanupOnlyPackagingService struct {
	cleanupFunc func(string) error
}

func (s *cleanupOnlyPackagingService) CleanupContext(_ context.Context, workspacePath string) error {
	if s.cleanupFunc != nil {
		return s.cleanupFunc(workspacePath)
	}
	return nil
}

func (s *cleanupOnlyPackagingService) BundleFilePathContext(context.Context, string, packaging.Target) (string, error) {
	return "", nil
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

func contextWithRequestMetadataForTest(ctx context.Context, requestID string, traceID string) context.Context {
	return platformhttp.ContextWithRequestMetadata(ctx, requestID, traceID)
}
