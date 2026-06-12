package catalogapi

import (
	"net/http"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/common"
	"skill_registry/app/observability"
)

func (s *Server) handleHealthz(writer http.ResponseWriter, request *http.Request) {
	observability.SetObservedAction(writer, string(auth.ActionReadHealthz))
	observability.SetObservedAuthDecision(writer, observability.AuthDecisionAllowed)
	common.WriteJSON(writer, http.StatusOK, map[string]any{
		"status": "ok",
	})
}

func (s *Server) handleReadyz(writer http.ResponseWriter, request *http.Request) {
	observability.SetObservedAction(writer, string(auth.ActionReadReadyz))
	observability.SetObservedAuthDecision(writer, observability.AuthDecisionAllowed)
	payload, status := s.systemModule.Readyz(request.Context())
	common.WriteJSON(writer, status, payload)
}

func (s *Server) handleMetrics(writer http.ResponseWriter, request *http.Request) {
	observability.SetObservedAction(writer, string(auth.ActionReadMetrics))
	observability.SetObservedAuthDecision(writer, observability.AuthDecisionAllowed)
	writer.Header().Set("Content-Type", "text/plain; version=0.0.4")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(s.metrics.RenderPrometheus()))
}

func (s *Server) handleMockRegistry(writer http.ResponseWriter, request *http.Request) {
	if s.mockRegistryModule == nil {
		http.NotFound(writer, request)
		return
	}
	s.mockRegistryModule.handleRequest(writer, request)
}
