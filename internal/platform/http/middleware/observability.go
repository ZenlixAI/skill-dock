package middleware

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	platformhttp "skill_registry/internal/platform/http"
	platformobservability "skill_registry/internal/platform/observability"
)

func WithObservability(metrics *platformobservability.Collector, logger *log.Logger) gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Request = platformhttp.EnsureObservedState(context.Request)
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

		state := platformhttp.ObservedStateFromContext(context.Request.Context())
		status := context.Writer.Status()
		outcome := platformhttp.ObservedOutcome(status, state)

		if metrics != nil {
			metrics.RecordRequest(method, path, status, time.Since(startedAt), state.Action, state.AuthDecision, outcome)
		}
		if logger != nil {
			fields := map[string]any{
				"event":         "http_request",
				"requestId":     platformhttp.RequestIDFromContext(context.Request.Context()),
				"traceId":       platformhttp.TraceIDFromContext(context.Request.Context()),
				"method":        method,
				"path":          path,
				"status":        status,
				"durationMs":    time.Since(startedAt).Milliseconds(),
				"role":          state.Principal.Role,
				"actor":         state.Principal.Actor,
				"authMechanism": state.Principal.AuthMechanism,
				"action":        state.Action,
				"authDecision":  state.AuthDecision,
				"outcome":       outcome,
			}
			if payload, err := json.Marshal(fields); err == nil {
				logger.Print(string(payload))
			}
		}
	}
}
