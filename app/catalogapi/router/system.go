package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerSystemRoutes(engine *gin.Engine, config Config) {
	engine.Any("/healthz", config.Healthz)
	engine.Any("/readyz", config.Readyz)
	engine.Any("/metrics", config.Metrics)

	if !config.DisableMockRegistry {
		engine.Any("/mock-npm", func(context *gin.Context) {
			context.Redirect(http.StatusMovedPermanently, "/mock-npm/")
		})
		engine.Any("/mock-npm/*path", config.MockRegistry)
	}
}
