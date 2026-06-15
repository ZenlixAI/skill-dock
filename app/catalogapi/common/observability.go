package common

import (
	"context"
	"net/http"
	"strings"

	"skill_registry/app/auth"
	"skill_registry/app/observability"
)

type ObservedState struct {
	Principal    auth.Principal
	Action       string
	AuthDecision observability.AuthDecision
	Outcome      string
}

type observedStateKey struct{}

func EnsureObservedState(request *http.Request) *http.Request {
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

func SetObservedPrincipal(request *http.Request, current auth.Principal) {
	state := ObservedStateFromContext(request.Context())
	if state == nil {
		return
	}
	state.Principal = current
}

func SetObservedAction(request *http.Request, action string) {
	state := ObservedStateFromContext(request.Context())
	if state == nil {
		return
	}
	state.Action = action
}

func SetObservedAuthDecision(request *http.Request, decision observability.AuthDecision) {
	state := ObservedStateFromContext(request.Context())
	if state == nil {
		return
	}
	state.AuthDecision = decision
}

func SetObservedOutcome(request *http.Request, outcome string) {
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
	if status == http.StatusUnauthorized {
		return "unauthorized"
	}
	if status == http.StatusForbidden {
		return "forbidden"
	}
	return "failed"
}
