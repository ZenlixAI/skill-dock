package middleware

import (
	stdhttp "net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

func WithCORS() gin.HandlerFunc {
	return func(context *gin.Context) {
		origin := strings.TrimSpace(context.GetHeader("Origin"))
		if origin != "" && isLocalOperatorOrigin(origin) {
			context.Header("Access-Control-Allow-Origin", origin)
			context.Header("Vary", "Origin")
			context.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Actor, X-Role, X-Trace-Id")
			context.Header("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
			context.Header("Access-Control-Allow-Credentials", "true")
			context.Header("Access-Control-Max-Age", "600")
		}

		if context.Request.Method == stdhttp.MethodOptions && origin != "" && isLocalOperatorOrigin(origin) {
			context.AbortWithStatus(stdhttp.StatusNoContent)
			return
		}

		context.Next()
	}
}

func isLocalOperatorOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "127.0.0.1" || host == "localhost"
}
