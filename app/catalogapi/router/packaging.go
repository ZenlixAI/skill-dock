package router

import "github.com/gin-gonic/gin"

func registerPackagingRoutes(engine *gin.Engine, config Config) {
	if !config.ServesWriteRoutes {
		return
	}

	group := engine.Group("/api/v1/skill-packaging")
	group.POST("/prepare", config.PackagingPrepare)
	group.POST("/validate", config.PackagingValidate)
	group.POST("/pack", config.PackagingPack)
	group.POST("/publish", config.PackagingPublish)
	group.POST("/cleanup", config.PackagingCleanup)
	group.GET("/bundle", config.PackagingBundle)
}
