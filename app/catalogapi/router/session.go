package router

import "github.com/gin-gonic/gin"

func registerSessionRoutes(engine *gin.Engine, config Config) {
	group := engine.Group("/api/v1/session")
	group.GET("", config.Session)
	group.POST("/login", config.SessionLogin)
	group.POST("/logout", config.SessionLogout)
}
