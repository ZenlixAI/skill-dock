package router

import "github.com/gin-gonic/gin"

func registerPackagingRoutes(engine *gin.Engine, config Config) {
	if !config.ServesWriteRoutes {
		return
	}

	engine.Any("/api/v1/skill-packaging/prepare", config.PackagingPrepare)
	engine.Any("/api/v1/skill-packaging/validate", config.PackagingValidate)
	engine.Any("/api/v1/skill-packaging/pack", config.PackagingPack)
	engine.Any("/api/v1/skill-packaging/publish", config.PackagingPublish)
	engine.Any("/api/v1/skill-packaging/cleanup", config.PackagingCleanup)
	engine.Any("/api/v1/skill-packaging/bundle", config.PackagingBundle)
}
