package catalogapi

import (
	"context"
	"net/http"
	"testing"

	"skill_registry/app/auth"
	"skill_registry/business/catalog"
)

func TestCatalogMutationModuleCreatesReleaseRecordAndRecordsCreatedAudit(t *testing.T) {
	store := &captureAuditStore{}
	module := &catalogMutationModule{
		repo:  catalog.NewMemoryRepository(catalog.SampleData()),
		audit: NewAuditRecorder(store),
	}
	principal := auth.Principal{
		Role:          "publisher",
		Actor:         "svc-publisher",
		AuthMechanism: "configured_token",
	}
	ctx := contextWithRequestMetadataForTest(context.Background(), "req_release", "trace_release")

	response, status, err := module.createReleaseRecord(ctx, principal, "@platform.skill/agent-notes", "0.1.0", nil)

	if err != nil {
		t.Fatalf("create release record: %v", err)
	}
	if status != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, status)
	}
	if !response.Created {
		t.Fatalf("expected created response, got %+v", response)
	}
	if len(store.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(store.events))
	}
	if store.events[0].Action != "release.record" || store.events[0].Outcome != "succeeded" {
		t.Fatalf("unexpected audit event: %+v", store.events[0])
	}
}
