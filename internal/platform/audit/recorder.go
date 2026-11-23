package audit

import (
	"context"
	"net/http"
	"strings"
	"time"

	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
	"skill_registry/internal/platform/observability"
	persist "skill_registry/internal/platform/persistence"
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
	event.RequestID = platformhttp.RequestIDFromContext(ctx)
	event.TraceID = platformhttp.TraceIDFromContext(ctx)
	event.Actor = current.Actor
	event.Role = current.Role
	event.AuthMechanism = current.AuthMechanism
	event.ActorType = actorType(current)
	event.ActorUserID = current.UserID
	event.ActorSubject = current.Subject
	event.ActorOrganizationID = current.TenantID
	if event.OrganizationID == "" {
		event.OrganizationID = current.TenantID
	}
	event.ActorDisplay = firstNonEmpty(current.DisplayName, current.Email, current.Actor)
	event.TokenID = current.TokenID
	event.ClientID = current.ClientID
	event.Endpoint = endpoint
	event.Method = method
	return r.store.Append(ctx, event)
}

func actorType(current auth.Principal) string {
	switch {
	case current.PlatformAdmin:
		return "platform_admin"
	case current.IsUser():
		return "user"
	case current.IsService():
		return "service"
	case current.IsAnonymous():
		return "anonymous"
	default:
		return "legacy"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
