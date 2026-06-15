package http

import "github.com/gin-gonic/gin"

type Engine = gin.Engine

// TestNewEngineHook is a narrow test hook used by app-level regression tests to
// verify the router constructs handlers via this shared seam.
var TestNewEngineHook func(*gin.Engine)

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
	if TestNewEngineHook != nil {
		TestNewEngineHook(engine)
	}
	return engine
}
