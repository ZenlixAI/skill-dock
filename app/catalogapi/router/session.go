package router

import "github.com/gin-gonic/gin"

func registerSessionRoutes(engine *gin.Engine, config Config) {
	engine.Any("/api/v1/session", config.Session)
	engine.Any("/api/v1/session/login", config.SessionLogin)
	engine.Any("/api/v1/session/logout", config.SessionLogout)
}
