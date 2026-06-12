package audit

import (
	"context"
	"net/http"
	"time"

	"skill_registry/app/catalogapi/common"

	"skill_registry/app/auth"
	"skill_registry/app/observability"
	"skill_registry/foundation/persist"
)

type Recorder struct {
	store persist.AuditStore
}

func NewRecorder(store persist.AuditStore) *Recorder {
	return &Recorder{store: store}
}

func (r *Recorder) RecordMutation(request *http.Request, action string, packageName string, version string, status int, outcome string, details map[string]any) error {
	return r.record(request.Context(), auth.PrincipalFromContext(request.Context()), request.Method, request.URL.EscapedPath(), persist.AuditEvent{
		Action:       action,
		PackageName:  packageName,
		Version:      new(version),
		Outcome:      outcome,
		StatusCode:   status,
		Details:      details,
		AuthDecision: observability.AuthDecisionAllowed,
		CreatedAt:    time.Now().UTC(),
	})
}

func (r *Recorder) RecordAuthorizationDenied(request *http.Request, current auth.Principal, outcome string, status int, packageName string, version string, details map[string]any) error {
	return r.record(request.Context(), current, request.Method, request.URL.EscapedPath(), persist.AuditEvent{
		Action:       "auth.denied",
		PackageName:  packageName,
		Version:      new(version),
		Outcome:      outcome,
		StatusCode:   status,
		Details:      details,
		AuthDecision: observability.AuthDecisionDenied,
		CreatedAt:    time.Now().UTC(),
	})
}

func (r *Recorder) RecordAuthenticationDenied(request *http.Request) error {
	return r.record(request.Context(), auth.Principal{}, request.Method, request.URL.EscapedPath(), persist.AuditEvent{
		Action:       "auth.denied",
		Outcome:      "unauthorized",
		StatusCode:   http.StatusUnauthorized,
		AuthDecision: observability.AuthDecisionDenied,
		CreatedAt:    time.Now().UTC(),
	})
}

func (r *Recorder) RecordMutationForEndpoint(ctx context.Context, current auth.Principal, method string, endpoint string, action string, packageName string, version string, status int, outcome string, details map[string]any) error {
	return r.record(ctx, current, method, endpoint, persist.AuditEvent{
		Action:       action,
		PackageName:  packageName,
		Version:      new(version),
		Outcome:      outcome,
		StatusCode:   status,
		Details:      details,
		AuthDecision: observability.AuthDecisionAllowed,
		CreatedAt:    time.Now().UTC(),
	})
}

func (r *Recorder) record(ctx context.Context, current auth.Principal, method string, endpoint string, event persist.AuditEvent) error {
	if r == nil || r.store == nil || ctx == nil {
		return nil
	}
	event.RequestID = common.RequestIDFromContext(ctx)
	event.TraceID = common.TraceIDFromContext(ctx)
	event.Actor = current.Actor
	event.Role = current.Role
	event.AuthMechanism = current.AuthMechanism
	event.Endpoint = endpoint
	event.Method = method
	return r.store.Append(ctx, event)
}
