package http

import (
	"context"
	stdhttp "net/http"

	"github.com/gin-gonic/gin"

	"skill_registry/app/catalogapi/common"
	"skill_registry/app/catalogapi/response"
	systemmodule "skill_registry/app/catalogapi/system"
	"skill_registry/internal/platform/auth"
	"skill_registry/internal/platform/observability"
)

type ReadyzFunc func(context.Context) (systemmodule.Readyz, int)
type MetricsFunc func() string

type Handlers struct {
	ReadyzFunc  ReadyzFunc
	MetricsFunc MetricsFunc
}

type Config struct {
	Healthz gin.HandlerFunc
	Readyz  gin.HandlerFunc
	Metrics gin.HandlerFunc
}

func New(readyz ReadyzFunc, metrics MetricsFunc) Handlers {
	return Handlers{
		ReadyzFunc:  readyz,
		MetricsFunc: metrics,
	}
}

func RegisterRoutes(engine *gin.Engine, config Config) {
	registerHeadCompatibleRoute(engine, "/healthz", config.Healthz)
	registerHeadCompatibleRoute(engine, "/readyz", config.Readyz)
	registerHeadCompatibleRoute(engine, "/metrics", config.Metrics)
}

func (h Handlers) Healthz(context *gin.Context) {
	common.SetObservedAction(context.Request, string(auth.ActionReadHealthz))
	common.SetObservedAuthDecision(context.Request, observability.AuthDecisionAllowed)
	response.WriteJSON(context, stdhttp.StatusOK, map[string]any{
		"status": "ok",
	})
}

func (h Handlers) Readyz(context *gin.Context) {
	common.SetObservedAction(context.Request, string(auth.ActionReadReadyz))
	common.SetObservedAuthDecision(context.Request, observability.AuthDecisionAllowed)

	if h.ReadyzFunc == nil {
		response.WriteJSON(context, stdhttp.StatusOK, systemmodule.Readyz{
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
	context.Data(stdhttp.StatusOK, "text/plain; version=0.0.4", []byte(body))
}

func registerHeadCompatibleRoute(engine *gin.Engine, path string, handler gin.HandlerFunc) {
	engine.Match([]string{stdhttp.MethodGet, stdhttp.MethodHead}, path, handler)
}
