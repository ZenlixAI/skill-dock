package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/common"
	"skill_registry/app/observability"
)

type observedState struct {
	principal    auth.Principal
	action       string
	authDecision observability.AuthDecision
	outcome      string
}

type observedStateKey struct{}

func WithObservability(metrics *observability.Collector, logger *log.Logger) gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Request = withObservedState(context.Request)
		startedAt := time.Now()

		context.Next()

		path := ""
		if context.Request != nil && context.Request.URL != nil {
			path = context.Request.URL.EscapedPath()
			if path == "" {
				path = context.Request.URL.Path
			}
		}
		method := ""
		if context.Request != nil {
			method = context.Request.Method
		}

		state := observedStateFromContext(context.Request.Context())
		status := context.Writer.Status()
		outcome := observedOutcome(status, state)

		if metrics != nil {
			metrics.RecordRequest(method, path, status, time.Since(startedAt), state.action, state.authDecision, outcome)
		}
		if logger != nil {
			fields := map[string]any{
				"event":         "http_request",
				"requestId":     common.RequestIDFromContext(context.Request.Context()),
				"traceId":       common.TraceIDFromContext(context.Request.Context()),
				"method":        method,
				"path":          path,
				"status":        status,
				"durationMs":    time.Since(startedAt).Milliseconds(),
				"role":          state.principal.Role,
				"actor":         state.principal.Actor,
				"authMechanism": state.principal.AuthMechanism,
				"action":        state.action,
				"authDecision":  state.authDecision,
				"outcome":       outcome,
			}
			if payload, err := json.Marshal(fields); err == nil {
				logger.Print(string(payload))
			}
		}
	}
}

func SetObservedPrincipal(request *http.Request, current auth.Principal) {
	state := observedStateFromContext(request.Context())
	if state == nil {
		return
	}
	state.principal = current
}

func SetObservedAction(request *http.Request, action string) {
	state := observedStateFromContext(request.Context())
	if state == nil {
		return
	}
	state.action = action
}

func SetObservedAuthDecision(request *http.Request, decision observability.AuthDecision) {
	state := observedStateFromContext(request.Context())
	if state == nil {
		return
	}
	state.authDecision = decision
}

func SetObservedOutcome(request *http.Request, outcome string) {
	state := observedStateFromContext(request.Context())
	if state == nil {
		return
	}
	state.outcome = outcome
}

func withObservedState(request *http.Request) *http.Request {
	if request == nil {
		return nil
	}
	if observedStateFromContext(request.Context()) != nil {
		return request
	}
	ctx := context.WithValue(request.Context(), observedStateKey{}, &observedState{})
	return request.WithContext(ctx)
}

func observedStateFromContext(ctx context.Context) *observedState {
	if ctx == nil {
		return nil
	}
	state, _ := ctx.Value(observedStateKey{}).(*observedState)
	return state
}

func observedOutcome(status int, state *observedState) string {
	if state != nil && strings.TrimSpace(state.outcome) != "" {
		return state.outcome
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
