package http

import "github.com/gin-gonic/gin"

type Engine = gin.Engine

func NewEngine(middleware []gin.HandlerFunc) *gin.Engine {
	engine := gin.New()
	engine.UseRawPath = true
	engine.UnescapePathValues = false
	engine.HandleMethodNotAllowed = true
	engine.RedirectTrailingSlash = false
	engine.RedirectFixedPath = false
	if len(middleware) > 0 {
		engine.Use(middleware...)
	}
	return engine
}
