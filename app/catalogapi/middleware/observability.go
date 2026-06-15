package middleware

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"skill_registry/app/catalogapi/common"
	"skill_registry/app/observability"
)

func WithObservability(metrics *observability.Collector, logger *log.Logger) gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Request = common.EnsureObservedState(context.Request)
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

		state := common.ObservedStateFromContext(context.Request.Context())
		status := context.Writer.Status()
		outcome := common.ObservedOutcome(status, state)

		if metrics != nil {
			metrics.RecordRequest(method, path, status, time.Since(startedAt), state.Action, state.AuthDecision, outcome)
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
