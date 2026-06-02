package catalogapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"skill_registry/app/auth"
	"skill_registry/app/observability"
	"skill_registry/foundation/persist"
)

func TestAuditRecorderRecordMutationIncludesPrincipalAndAuthDecision(t *testing.T) {
	store := &captureAuditStore{}
	recorder := NewAuditRecorder(store)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/skills/%40platform.skill%2Fagent-notes/versions/0.1.0/release-record", nil)
	ctx := context.WithValue(request.Context(), requestIDContextKey, "req_123")
	ctx = context.WithValue(ctx, traceIDContextKey, "trace_123")
	ctx = auth.ContextWithPrincipal(ctx, auth.Principal{
		Role:          "publisher",
		Actor:         "svc-publisher",
		AuthMechanism: "configured_token",
	})
	request = request.WithContext(ctx)

	recorder.RecordMutation(request, "release.record", "@platform.skill/agent-notes", "0.1.0", http.StatusCreated, "succeeded", map[string]any{
		"created": true,
	})

	if len(store.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(store.events))
	}
	event := store.events[0]
	if event.AuthDecision != observability.AuthDecisionAllowed {
		t.Fatalf("expected allowed auth decision, got %s", event.AuthDecision)
	}
	if event.Actor != "svc-publisher" || event.Role != "publisher" {
		t.Fatalf("unexpected principal in audit event: %+v", event)
	}
	if event.RequestID != "req_123" || event.TraceID != "trace_123" {
		t.Fatalf("unexpected request metadata: %+v", event)
	}
}

func TestAuditRecorderRecordAuthorizationDeniedIncludesDecisionAndCoordinates(t *testing.T) {
	store := &captureAuditStore{}
	recorder := NewAuditRecorder(store)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/skills/%40platform.skill%2Fpdf-skill/governance-changes", nil)
	ctx := context.WithValue(request.Context(), requestIDContextKey, "req_456")
	ctx = context.WithValue(ctx, traceIDContextKey, "trace_456")
	request = request.WithContext(ctx)

	recorder.RecordAuthorizationDenied(request, auth.Principal{
		Role:          "reader",
		Actor:         "reader",
		AuthMechanism: "configured_token",
	}, "package_forbidden", http.StatusForbidden, "@platform.skill/pdf-skill", "", map[string]any{
		"packageName": "@platform.skill/pdf-skill",
	})

	if len(store.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(store.events))
	}
	event := store.events[0]
	if event.Action != "auth.denied" {
		t.Fatalf("expected auth.denied action, got %s", event.Action)
	}
	if event.AuthDecision != observability.AuthDecisionDenied {
		t.Fatalf("expected denied auth decision, got %s", event.AuthDecision)
	}
	if event.PackageName != "@platform.skill/pdf-skill" {
		t.Fatalf("unexpected package name: %+v", event)
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
