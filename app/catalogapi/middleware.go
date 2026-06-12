package catalogapi

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/common"
	"skill_registry/app/observability"
	"skill_registry/business/catalog"
)

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		origin := strings.TrimSpace(request.Header.Get("Origin"))
		if origin != "" && isLocalOperatorOrigin(origin) {
			writer.Header().Set("Access-Control-Allow-Origin", origin)
			writer.Header().Set("Vary", "Origin")
			writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Actor, X-Role, X-Trace-Id")
			writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
			writer.Header().Set("Access-Control-Allow-Credentials", "true")
			writer.Header().Set("Access-Control-Max-Age", "600")
		}

		if request.Method == http.MethodOptions && origin != "" && isLocalOperatorOrigin(origin) {
			writer.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(writer, request)
	})
}

func isLocalOperatorOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "127.0.0.1" || host == "localhost"
}

func (s *Server) withRequestMetadata(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestID := "req_" + strconv.FormatUint(s.requestID.Add(1), 10)
		writer.Header().Set("X-Request-Id", requestID)
		traceID := strings.TrimSpace(request.Header.Get("X-Trace-Id"))
		if traceID == "" {
			traceID = "trace_" + strings.TrimPrefix(requestID, "req_")
		}
		writer.Header().Set("X-Trace-Id", traceID)
		ctx := common.ContextWithRequestMetadata(request.Context(), requestID, traceID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

func (s *Server) withObservability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		startedAt := time.Now()
		recorder := observability.NewStatusRecorder(writer)
		next.ServeHTTP(recorder, request)
		duration := time.Since(startedAt)
		s.metrics.RecordRequest(
			request.Method,
			request.URL.EscapedPath(),
			recorder.Status,
			duration,
			recorder.ObservedAction,
			recorder.ObservedAuthDecision,
			recorder.Outcome(),
		)
		s.logRequest(request, recorder, startedAt)
	})
}

func (s *Server) requireAction(next http.HandlerFunc, action auth.Action) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		observability.SetObservedAction(writer, string(action))
		authPrincipal, ok := s.authenticateRequest(request)
		if !ok {
			observability.SetObservedAuthDecision(writer, observability.AuthDecisionDenied)
			observability.SetObservedOutcome(writer, "unauthorized")
			common.WriteCatalogError(writer, request, catalog.NewError(401, "unauthorized", "Authentication is required.", nil))
			s.recordAuthenticationDeniedAudit(request)
			return
		}
		request = request.WithContext(auth.ContextWithPrincipal(request.Context(), authPrincipal))
		observability.SetObservedPrincipal(writer, authPrincipal)
		if !s.accessWorkflow.authorizeAction(writer, request, action, "", "") {
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (s *Server) authenticateRequest(request *http.Request) (auth.Principal, bool) {
	if current, ok := s.authorizer.Authenticate(request); ok {
		return current, true
	}
	if current, ok := s.sessionManager.Authenticate(request); ok {
		return current, true
	}
	return auth.Principal{}, false
}

func (s *Server) logRequest(request *http.Request, recorder *observability.StatusRecorder, startedAt time.Time) {
	fields := map[string]any{
		"event":         "http_request",
		"requestId":     common.RequestIDFromContext(request.Context()),
		"traceId":       common.TraceIDFromContext(request.Context()),
		"method":        request.Method,
		"path":          request.URL.EscapedPath(),
		"status":        recorder.Status,
		"durationMs":    time.Since(startedAt).Milliseconds(),
		"role":          recorder.ObservedPrincipal.Role,
		"actor":         recorder.ObservedPrincipal.Actor,
		"authMechanism": recorder.ObservedPrincipal.AuthMechanism,
		"action":        recorder.ObservedAction,
		"authDecision":  recorder.ObservedAuthDecision,
		"outcome":       recorder.Outcome(),
	}
	if payload, err := json.Marshal(fields); err == nil {
		s.logger.Print(string(payload))
	}
}
