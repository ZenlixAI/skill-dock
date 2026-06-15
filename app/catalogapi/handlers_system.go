package catalogapi

import (
	"net/http"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/common"
	"skill_registry/app/observability"
)

func (s *Server) handleHealthz(writer http.ResponseWriter, request *http.Request) {
	common.SetObservedAction(request, string(auth.ActionReadHealthz))
	common.SetObservedAuthDecision(request, observability.AuthDecisionAllowed)
	common.WriteJSON(writer, http.StatusOK, map[string]any{
		"status": "ok",
	})
}

func (s *Server) handleReadyz(writer http.ResponseWriter, request *http.Request) {
	common.SetObservedAction(request, string(auth.ActionReadReadyz))
	common.SetObservedAuthDecision(request, observability.AuthDecisionAllowed)
	payload, status := s.systemModule.Readyz(request.Context())
	common.WriteJSON(writer, status, payload)
}

func (s *Server) handleMetrics(writer http.ResponseWriter, request *http.Request) {
	common.SetObservedAction(request, string(auth.ActionReadMetrics))
	common.SetObservedAuthDecision(request, observability.AuthDecisionAllowed)
	writer.Header().Set("Content-Type", "text/plain; version=0.0.4")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(s.metrics.RenderPrometheus()))
}
