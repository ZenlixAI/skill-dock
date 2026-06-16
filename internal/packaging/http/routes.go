package http

import "github.com/gin-gonic/gin"

type Config struct {
	ServesWriteRoutes bool
	ImportReview      gin.HandlerFunc
	Publish           gin.HandlerFunc
}

func RegisterRoutes(engine *gin.Engine, config Config) {
	if !config.ServesWriteRoutes {
		return
	}

	group := engine.Group("/api/v1/skill-packaging")
	group.POST("/import-review", config.ImportReview)
	group.POST("/publish", config.Publish)
}
