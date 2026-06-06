package catalogapi

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"skill_registry/app/auth"
	"skill_registry/app/observability"
)

func TestAccessWorkflowDeniesPackageAccessAndRecordsAudit(t *testing.T) {
	store := &captureAuditStore{}
	workflow := &accessWorkflow{
		authorizer: auth.NewAuthorizer(auth.Config{}),
		audit:      NewAuditRecorder(store),
		logger:     log.New(io.Discard, "", 0),
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills?package_name=%40platform.skill%2Fprivate-skill", nil)
	ctx := auth.ContextWithPrincipal(request.Context(), auth.Principal{
		Role:            "reader",
		Actor:           "svc-reader",
		AuthMechanism:   "configured_token",
		AllowedPackages: []string{"@platform.skill/other-skill"},
	})
	request = request.WithContext(contextWithRequestMetadataForTest(ctx, "req_denied", "trace_denied"))
	recorder := httptest.NewRecorder()

	ok := workflow.authorizePackage(recorder, request, "@platform.skill/private-skill")

	if ok {
		t.Fatal("expected package access to be denied")
	}
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusForbidden, recorder.Code, recorder.Body.String())
	}
	if len(store.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(store.events))
	}
	if store.events[0].Action != "auth.denied" {
		t.Fatalf("expected auth.denied action, got %+v", store.events[0])
	}
	if store.events[0].AuthDecision != observability.AuthDecisionDenied {
		t.Fatalf("expected denied auth decision, got %+v", store.events[0])
	}
}
