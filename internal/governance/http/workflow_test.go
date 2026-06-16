package http

import (
	"context"
	"net/http"
	"testing"

	cataloginfra "skill_registry/internal/catalog/infra"
	platformaudit "skill_registry/internal/platform/audit"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
	persist "skill_registry/internal/platform/persistence"
)

func TestWorkflowCreatesReleaseRecordAndRecordsCreatedAudit(t *testing.T) {
	store := &captureAuditStore{}
	workflow := NewWorkflow(
		auth.NewAuthorizer(auth.Config{}),
		cataloginfra.NewMemoryRepository(cataloginfra.SampleData()),
		cataloginfra.NewMemoryRepository(cataloginfra.SampleData()),
		platformaudit.NewRecorder(store),
	)
	principal := auth.Principal{
		Role:          "publisher",
		Actor:         "svc-publisher",
		AuthMechanism: "configured_token",
	}
	ctx := contextWithRequestMetadataForTest(context.Background(), "req_release", "trace_release")

	response, status, err := workflow.CreateReleaseRecord(ctx, principal, "@platform.skill/agent-notes", "0.1.0", nil)
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

func TestWorkflowAppliesSkillGovernanceAndRecordsAudit(t *testing.T) {
	store := &captureAuditStore{}
	workflow := NewWorkflow(
		auth.NewAuthorizer(auth.Config{}),
		cataloginfra.NewMemoryRepository(cataloginfra.SampleData()),
		cataloginfra.NewMemoryRepository(cataloginfra.SampleData()),
		platformaudit.NewRecorder(store),
	)
	principal := auth.Principal{
		Role:          "publisher",
		Actor:         "user_789",
		AuthMechanism: "configured_token",
	}
	ctx := contextWithRequestMetadataForTest(context.Background(), "req_skill_governance", "trace_skill_governance")

	response, status, err := workflow.ApplySkillGovernanceChange(ctx, principal, "@platform.skill/code-review", []byte(`{"action":"retire","reason":"merged into new pack"}`))
	if err != nil {
		t.Fatalf("apply skill governance: %v", err)
	}
	if status != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, status)
	}
	if response.Record.ChangedBy != "user_789" {
		t.Fatalf("expected actor user_789, got %+v", response.Record)
	}
	if len(store.events) != 1 || store.events[0].Action != "governance.skill" {
		t.Fatalf("unexpected audit events: %+v", store.events)
	}
}

func TestWorkflowAppliesVersionGovernanceAndRecordsAudit(t *testing.T) {
	store := &captureAuditStore{}
	workflow := NewWorkflow(
		auth.NewAuthorizer(auth.Config{}),
		cataloginfra.NewMemoryRepository(cataloginfra.SampleData()),
		cataloginfra.NewMemoryRepository(cataloginfra.SampleData()),
		platformaudit.NewRecorder(store),
	)
	principal := auth.Principal{
		Role:          "publisher",
		Actor:         "user_123",
		AuthMechanism: "configured_token",
	}
	ctx := contextWithRequestMetadataForTest(context.Background(), "req_version_governance", "trace_version_governance")

	response, status, err := workflow.ApplyVersionGovernanceChange(ctx, principal, "@platform.skill/pdf-skill", "1.2.0", []byte(`{"action":"yank","reason":"security issue","replacementRecommendation":"1.1.0"}`))
	if err != nil {
		t.Fatalf("apply version governance: %v", err)
	}
	if status != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, status)
	}
	if response.Record.ChangedBy != "user_123" {
		t.Fatalf("expected actor user_123, got %+v", response.Record)
	}
	if len(store.events) != 1 || store.events[0].Action != "governance.version" {
		t.Fatalf("unexpected audit events: %+v", store.events)
	}
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
