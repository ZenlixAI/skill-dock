package catalogapi

import (
	"context"
	"net/http"
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
