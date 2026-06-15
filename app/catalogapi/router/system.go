package router

import "github.com/gin-gonic/gin"

func registerSystemRoutes(engine *gin.Engine, config Config) {
	engine.Any("/healthz", config.Healthz)
	engine.Any("/readyz", config.Readyz)
	engine.Any("/metrics", config.Metrics)

	if !config.DisableMockRegistry {
		engine.Any("/mock-npm/*path", config.MockRegistry)
	}
}
