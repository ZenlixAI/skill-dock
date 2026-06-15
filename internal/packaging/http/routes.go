package http

import "github.com/gin-gonic/gin"

type Config struct {
	ServesWriteRoutes bool
	Prepare           gin.HandlerFunc
	Validate          gin.HandlerFunc
	Pack              gin.HandlerFunc
	Publish           gin.HandlerFunc
	Cleanup           gin.HandlerFunc
	Bundle            gin.HandlerFunc
}

func RegisterRoutes(engine *gin.Engine, config Config) {
	if !config.ServesWriteRoutes {
		return
	}

	group := engine.Group("/api/v1/skill-packaging")
	group.POST("/prepare", config.Prepare)
	group.POST("/validate", config.Validate)
	group.POST("/pack", config.Pack)
	group.POST("/publish", config.Publish)
	group.POST("/cleanup", config.Cleanup)
	group.GET("/bundle", config.Bundle)
}
