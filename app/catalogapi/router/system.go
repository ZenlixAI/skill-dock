package router

import "github.com/gin-gonic/gin"

func registerSystemRoutes(engine *gin.Engine, config Config) {
	registerHeadCompatibleRoute(engine, "/healthz", config.Healthz)
	registerHeadCompatibleRoute(engine, "/readyz", config.Readyz)
	registerHeadCompatibleRoute(engine, "/metrics", config.Metrics)
}
