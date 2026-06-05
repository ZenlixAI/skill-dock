package catalogapi

import (
	"context"
	"net/http"
	"time"

	"skill_registry/app/auth"
	"skill_registry/app/observability"
	"skill_registry/foundation/persist"
)

type AuditRecorder struct {
	store persist.AuditStore
}

func NewAuditRecorder(store persist.AuditStore) *AuditRecorder {
	return &AuditRecorder{store: store}
}

func (r *AuditRecorder) RecordMutation(request *http.Request, action string, packageName string, version string, status int, outcome string, details map[string]any) error {
	return r.record(request.Context(), auth.PrincipalFromContext(request.Context()), request.Method, request.URL.EscapedPath(), persist.AuditEvent{
		Action:       action,
		PackageName:  packageName,
		Version:      stringPtr(version),
		Outcome:      outcome,
		StatusCode:   status,
		Details:      details,
		AuthDecision: observability.AuthDecisionAllowed,
		CreatedAt:    time.Now().UTC(),
	})
}

func (r *AuditRecorder) RecordAuthorizationDenied(request *http.Request, current auth.Principal, outcome string, status int, packageName string, version string, details map[string]any) error {
	return r.record(request.Context(), current, request.Method, request.URL.EscapedPath(), persist.AuditEvent{
		Action:       "auth.denied",
		PackageName:  packageName,
		Version:      stringPtr(version),
		Outcome:      outcome,
		StatusCode:   status,
		Details:      details,
		AuthDecision: observability.AuthDecisionDenied,
		CreatedAt:    time.Now().UTC(),
	})
}

func (r *AuditRecorder) RecordAuthenticationDenied(request *http.Request) error {
	return r.record(request.Context(), auth.Principal{}, request.Method, request.URL.EscapedPath(), persist.AuditEvent{
		Action:       "auth.denied",
		Outcome:      "unauthorized",
		StatusCode:   http.StatusUnauthorized,
		AuthDecision: observability.AuthDecisionDenied,
		CreatedAt:    time.Now().UTC(),
	})
}

func (r *AuditRecorder) RecordMutationForEndpoint(ctx context.Context, current auth.Principal, method string, endpoint string, action string, packageName string, version string, status int, outcome string, details map[string]any) error {
	return r.record(ctx, current, method, endpoint, persist.AuditEvent{
		Action:       action,
		PackageName:  packageName,
		Version:      stringPtr(version),
		Outcome:      outcome,
		StatusCode:   status,
		Details:      details,
		AuthDecision: observability.AuthDecisionAllowed,
		CreatedAt:    time.Now().UTC(),
	})
}

func (r *AuditRecorder) record(ctx context.Context, current auth.Principal, method string, endpoint string, event persist.AuditEvent) error {
	if r == nil || r.store == nil || ctx == nil {
		return nil
	}
	event.RequestID = requestIDFromContext(ctx)
	event.TraceID = traceIDFromContext(ctx)
	event.Actor = current.Actor
	event.Role = current.Role
	event.AuthMechanism = current.AuthMechanism
	event.Endpoint = endpoint
	event.Method = method
	return r.store.Append(ctx, event)
}
