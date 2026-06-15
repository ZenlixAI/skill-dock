package systemhandlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/common"
	"skill_registry/app/catalogapi/response"
	systemmodule "skill_registry/app/catalogapi/system"
	"skill_registry/app/observability"
)

type ReadyzFunc func(context.Context) (systemmodule.Readyz, int)
type MetricsFunc func() string

type Handlers struct {
	ReadyzFunc  ReadyzFunc
	MetricsFunc MetricsFunc
}

func New(readyz ReadyzFunc, metrics MetricsFunc) Handlers {
	return Handlers{
		ReadyzFunc:  readyz,
		MetricsFunc: metrics,
	}
}

func (h Handlers) Healthz(context *gin.Context) {
	common.SetObservedAction(context.Request, string(auth.ActionReadHealthz))
	common.SetObservedAuthDecision(context.Request, observability.AuthDecisionAllowed)
	response.WriteJSON(context, http.StatusOK, map[string]any{
		"status": "ok",
	})
}

func (h Handlers) Readyz(context *gin.Context) {
	common.SetObservedAction(context.Request, string(auth.ActionReadReadyz))
	common.SetObservedAuthDecision(context.Request, observability.AuthDecisionAllowed)

	if h.ReadyzFunc == nil {
		response.WriteJSON(context, http.StatusOK, systemmodule.Readyz{
			Status: "ready",
			Checks: map[string]string{},
		})
		return
	}

	payload, status := h.ReadyzFunc(context.Request.Context())
	response.WriteJSON(context, status, payload)
}

func (h Handlers) Metrics(context *gin.Context) {
	common.SetObservedAction(context.Request, string(auth.ActionReadMetrics))
	common.SetObservedAuthDecision(context.Request, observability.AuthDecisionAllowed)

	body := ""
	if h.MetricsFunc != nil {
		body = h.MetricsFunc()
	}
	context.Data(http.StatusOK, "text/plain; version=0.0.4", []byte(body))
}
