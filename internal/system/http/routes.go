package http

import (
	"context"
	stdhttp "net/http"

	"github.com/gin-gonic/gin"

	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
	"skill_registry/internal/platform/observability"
)

type Readyz struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

type ReadyzFunc func(context.Context) (Readyz, int)
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
	platformhttp.SetObservedAction(context.Request, string(auth.ActionReadHealthz))
	platformhttp.SetObservedAuthDecision(context.Request, observability.AuthDecisionAllowed)
	platformhttp.WriteJSON(context.Writer, stdhttp.StatusOK, map[string]any{
		"status": "ok",
	})
}

func (h Handlers) Readyz(context *gin.Context) {
	platformhttp.SetObservedAction(context.Request, string(auth.ActionReadReadyz))
	platformhttp.SetObservedAuthDecision(context.Request, observability.AuthDecisionAllowed)

	if h.ReadyzFunc == nil {
		platformhttp.WriteJSON(context.Writer, stdhttp.StatusOK, Readyz{
			Status: "ready",
			Checks: map[string]string{},
		})
		return
	}

	payload, status := h.ReadyzFunc(context.Request.Context())
	platformhttp.WriteJSON(context.Writer, status, payload)
}

func (h Handlers) Metrics(context *gin.Context) {
	platformhttp.SetObservedAction(context.Request, string(auth.ActionReadMetrics))
	platformhttp.SetObservedAuthDecision(context.Request, observability.AuthDecisionAllowed)

	body := ""
	if h.MetricsFunc != nil {
		body = h.MetricsFunc()
	}
	context.Data(stdhttp.StatusOK, "text/plain; version=0.0.4", []byte(body))
}

func registerHeadCompatibleRoute(engine *gin.Engine, path string, handler gin.HandlerFunc) {
	engine.Match([]string{stdhttp.MethodGet, stdhttp.MethodHead}, path, handler)
}
