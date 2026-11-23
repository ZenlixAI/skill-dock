package httpserver

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(context *gin.Context) {
		path := context.Request.URL.Path
		if strings.HasPrefix(path, "/api/skills-sh/") || strings.HasPrefix(path, "/api/v1/public/") {
			context.Next()
			return
		}
		origin := strings.TrimSpace(context.GetHeader("Origin"))
		if origin != "" && s.originAllowed(origin) {
			context.Header("Access-Control-Allow-Origin", origin)
			context.Header("Vary", "Origin")
			context.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			context.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
			context.Header("Access-Control-Max-Age", "600")
		}
		if context.Request.Method == http.MethodOptions {
			context.Status(http.StatusNoContent)
			context.Abort()
			return
		}
		context.Next()
	}
}

func (s *Server) originAllowed(origin string) bool {
	if s == nil {
		return false
	}
	for _, allowed := range s.corsOrigins {
		if strings.EqualFold(strings.TrimSpace(allowed), origin) {
			return true
		}
	}
	return false
}
