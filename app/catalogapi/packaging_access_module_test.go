package catalogapi

import (
	"context"
	"net/http"
	"os"
	"testing"

	"skill_registry/app/auth"
	"skill_registry/business/packaging"
)

func TestPackagingAccessModuleDownloadsBundleAndRecordsAudit(t *testing.T) {
	service := packaging.NewService(t.TempDir())
	workspacePath, _ := prepareManagedWorkspaceForTest(t, service, "@platform.skill/fixture-skill", "0.2.0")
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
	ctx := contextWithRequestMetadataForTest(context.Background(), "req_bundle", "trace_bundle")

	content, err := module.downloadBundle(ctx, principal, workspacePath, "@platform.skill/fixture-skill", "0.2.0")

	if err != nil {
		t.Fatalf("download bundle: %v", err)
	}
	if len(content) == 0 {
		t.Fatal("expected bundle content")
	}
	if len(store.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(store.events))
	}
	if store.events[0].Action != "packaging.bundle.download" || store.events[0].StatusCode != http.StatusOK {
		t.Fatalf("unexpected audit event: %+v", store.events[0])
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
