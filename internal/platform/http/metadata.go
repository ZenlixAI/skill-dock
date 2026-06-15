package http

import (
	"context"
	stdhttp "net/http"
	"strings"

	platformauth "skill_registry/internal/platform/auth"
	platformobservability "skill_registry/internal/platform/observability"
)

type contextKey string

const (
	requestIDContextKey contextKey = "request_id"
	traceIDContextKey   contextKey = "trace_id"
)

func ContextWithRequestMetadata(ctx context.Context, requestID string, traceID string) context.Context {
	ctx = context.WithValue(ctx, requestIDContextKey, requestID)
	return context.WithValue(ctx, traceIDContextKey, traceID)
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey).(string)
	return requestID
}

func TraceIDFromContext(ctx context.Context) string {
	traceID, _ := ctx.Value(traceIDContextKey).(string)
	return traceID
}

type ObservedState struct {
	Principal    platformauth.Principal
	Action       string
	AuthDecision platformobservability.AuthDecision
	Outcome      string
}

type observedStateKey struct{}

func EnsureObservedState(request *stdhttp.Request) *stdhttp.Request {
	if request == nil {
		return nil
	}
	if ObservedStateFromContext(request.Context()) != nil {
		return request
	}
	ctx := context.WithValue(request.Context(), observedStateKey{}, &ObservedState{})
	return request.WithContext(ctx)
}

func ObservedStateFromContext(ctx context.Context) *ObservedState {
	if ctx == nil {
		return nil
	}
	state, _ := ctx.Value(observedStateKey{}).(*ObservedState)
	return state
}

func SetObservedPrincipal(request *stdhttp.Request, current platformauth.Principal) {
	state := ObservedStateFromContext(request.Context())
	if state == nil {
		return
	}
	state.Principal = current
}

func SetObservedAction(request *stdhttp.Request, action string) {
	state := ObservedStateFromContext(request.Context())
	if state == nil {
		return
	}
	state.Action = action
}

func SetObservedAuthDecision(request *stdhttp.Request, decision platformobservability.AuthDecision) {
	state := ObservedStateFromContext(request.Context())
	if state == nil {
		return
	}
	state.AuthDecision = decision
}

func SetObservedOutcome(request *stdhttp.Request, outcome string) {
	state := ObservedStateFromContext(request.Context())
	if state == nil {
		return
	}
	state.Outcome = outcome
}

func ObservedOutcome(status int, state *ObservedState) string {
	if state != nil && strings.TrimSpace(state.Outcome) != "" {
		return state.Outcome
	}
	if status >= 200 && status < 400 {
		return "succeeded"
	}
	if status == stdhttp.StatusUnauthorized {
		return "unauthorized"
	}
	if status == stdhttp.StatusForbidden {
		return "forbidden"
	}
	return "failed"
}
