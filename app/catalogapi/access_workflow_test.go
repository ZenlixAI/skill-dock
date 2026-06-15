package catalogapi

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"skill_registry/app/catalogapi/audit"
	"skill_registry/app/catalogapi/common"
	catalogmiddleware "skill_registry/app/catalogapi/middleware"
	"skill_registry/internal/platform/auth"
	"skill_registry/internal/platform/observability"
)

func TestAccessWorkflowDeniesPackageAccessAndRecordsAudit(t *testing.T) {
	store := &captureAuditStore{}
	workflow := &accessWorkflow{
		authorizer: auth.NewAuthorizer(auth.Config{}),
		audit:      audit.NewRecorder(store),
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

func TestAccessWorkflowAuthorizeActionFeedsGinObservability(t *testing.T) {
	var logBuffer bytes.Buffer
	workflow := &accessWorkflow{
		authorizer: auth.NewAuthorizer(auth.Config{}),
		audit:      audit.NewRecorder(&captureAuditStore{}),
		logger:     log.New(io.Discard, "", 0),
	}

	engine := gin.New()
	engine.Use(
		catalogmiddleware.WithRequestMetadata(nil),
		catalogmiddleware.WithObservability(nil, log.New(&logBuffer, "", 0)),
		catalogmiddleware.WithAuthentication(func(request *http.Request) (auth.Principal, bool) {
			return auth.Principal{
				Role:          "reader",
				Actor:         "svc-reader",
				AuthMechanism: "configured_token",
			}, true
		}),
	)
	engine.GET("/authorized", func(context *gin.Context) {
		if !workflow.authorizeAction(context.Writer, context.Request, auth.ActionReadCatalog, "", "") {
			return
		}
		context.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/authorized", nil)
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNoContent, recorder.Code, recorder.Body.String())
	}

	payload := findLoggedEvent(t, logBuffer.String(), "http_request")
	if payload["action"] != string(auth.ActionReadCatalog) {
		t.Fatalf("expected action %q, got %+v", auth.ActionReadCatalog, payload)
	}
	if payload["authDecision"] != string(observability.AuthDecisionAllowed) {
		t.Fatalf("expected authDecision %q, got %+v", observability.AuthDecisionAllowed, payload)
	}
	if payload["outcome"] != "succeeded" {
		t.Fatalf("expected succeeded outcome, got %+v", payload)
	}
	if payload["role"] != "reader" {
		t.Fatalf("expected role reader, got %+v", payload)
	}
}

func TestAccessWorkflowAuthorizeActionSetsNeutralObservedState(t *testing.T) {
	workflow := &accessWorkflow{
		authorizer: auth.NewAuthorizer(auth.Config{}),
		audit:      audit.NewRecorder(&captureAuditStore{}),
		logger:     log.New(io.Discard, "", 0),
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	request = common.EnsureObservedState(request)
	request = request.WithContext(auth.ContextWithPrincipal(request.Context(), auth.Principal{
		Role:          "reader",
		Actor:         "svc-reader",
		AuthMechanism: "configured_token",
	}))
	recorder := httptest.NewRecorder()

	ok := workflow.authorizeAction(recorder, request, auth.ActionReadCatalog, "", "")
	if !ok {
		t.Fatal("expected action to be authorized")
	}

	state := common.ObservedStateFromContext(request.Context())
	if state == nil {
		t.Fatal("expected observed state in request context")
	}
	if state.Action != string(auth.ActionReadCatalog) {
		t.Fatalf("expected action %q, got %+v", auth.ActionReadCatalog, state)
	}
	if state.AuthDecision != observability.AuthDecisionAllowed {
		t.Fatalf("expected authDecision %q, got %+v", observability.AuthDecisionAllowed, state)
	}
}
