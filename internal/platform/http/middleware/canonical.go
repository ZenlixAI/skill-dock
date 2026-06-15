package middleware

import (
	stdhttp "net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

func WithCanonicalRedirect() gin.HandlerFunc {
	return func(context *gin.Context) {
		if target, ok := canonicalRouteTarget(context.Request); ok {
			context.Redirect(stdhttp.StatusTemporaryRedirect, target)
			context.Abort()
			return
		}
		context.Next()
	}
}

func canonicalRouteTarget(request *stdhttp.Request) (string, bool) {
	if request == nil || request.URL == nil || request.Method == stdhttp.MethodConnect {
		return "", false
	}

	escapedPath := request.URL.EscapedPath()
	if escapedPath == "" {
		escapedPath = request.URL.Path
	}
	cleanedPath := cleanCanonicalPath(escapedPath)
	if cleanedPath != escapedPath {
		return canonicalRedirectTarget(cleanedPath, request.URL.RawQuery), true
	}
	return "", false
}

func cleanCanonicalPath(rawPath string) string {
	if rawPath == "" {
		return "/"
	}
	if rawPath[0] != '/' {
		rawPath = "/" + rawPath
	}
	cleanedPath := path.Clean(rawPath)
	if rawPath[len(rawPath)-1] == '/' && cleanedPath != "/" {
		if len(rawPath) == len(cleanedPath)+1 && strings.HasPrefix(rawPath, cleanedPath) {
			cleanedPath = rawPath
		} else {
			cleanedPath += "/"
		}
	}
	return cleanedPath
}

func canonicalRedirectTarget(path string, rawQuery string) string {
	if strings.TrimSpace(rawQuery) == "" {
		return path
	}
	return path + "?" + rawQuery
}
