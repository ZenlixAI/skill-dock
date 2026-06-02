package catalogapi

import (
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
	return r.record(request, auth.PrincipalFromContext(request.Context()), persist.AuditEvent{
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
	return r.record(request, current, persist.AuditEvent{
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
	return r.record(request, auth.Principal{}, persist.AuditEvent{
		Action:       "auth.denied",
		Outcome:      "unauthorized",
		StatusCode:   http.StatusUnauthorized,
		AuthDecision: observability.AuthDecisionDenied,
		CreatedAt:    time.Now().UTC(),
	})
}

func (r *AuditRecorder) record(request *http.Request, current auth.Principal, event persist.AuditEvent) error {
	if r == nil || r.store == nil || request == nil {
		return nil
	}
	event.RequestID = requestIDFromContext(request.Context())
	event.TraceID = traceIDFromContext(request.Context())
	event.Actor = current.Actor
	event.Role = current.Role
	event.AuthMechanism = current.AuthMechanism
	event.Endpoint = request.URL.EscapedPath()
	event.Method = request.Method
	return r.store.Append(request.Context(), event)
}
