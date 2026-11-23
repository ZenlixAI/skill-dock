package httpserver

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"skill_registry/internal/access"
	catalog "skill_registry/internal/catalog/domain"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
)

func (s *Server) authenticatedArtifactProxy(next http.Handler, privateNext http.Handler) gin.HandlerFunc {
	return func(context *gin.Context) {
		selected := next
		if context.Request.Method == http.MethodOptions {
			context.Status(http.StatusNoContent)
			return
		}
		if context.Request.Method != http.MethodGet && context.Request.Method != http.MethodHead {
			current := auth.PrincipalFromContext(context.Request.Context())
			if !current.PlatformAdmin && !(current.IsService() && (current.Role == "publisher" || current.Role == "operator")) {
				platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusForbidden, "forbidden", "Publishing through the artifact proxy requires a publisher service principal.", nil))
				return
			}
			selected.ServeHTTP(context.Writer, context.Request)
			return
		}
		packageName, ok := packageFromArtifactPath(context.Request.URL.EscapedPath())
		if !ok {
			// Root and non-package paths keep the legacy proxy behavior.
			selected.ServeHTTP(context.Writer, context.Request)
			return
		}
		if s == nil || s.accessStore == nil || s.access == nil {
			selected.ServeHTTP(context.Writer, context.Request)
			return
		}
		record, err := s.accessStore.GetSkillAccess(context.Request.Context(), packageName)
		if errors.Is(err, access.ErrNotFound) {
			selected.ServeHTTP(context.Writer, context.Request)
			return
		}
		if err != nil {
			platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusInternalServerError, "internal_error", "Internal server error.", nil))
			return
		}
		if record.Visibility == "" || record.Visibility == "public" {
			selected.ServeHTTP(context.Writer, context.Request)
			return
		}

		current := auth.PrincipalFromContext(context.Request.Context())
		if current.IsAnonymous() {
			platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusUnauthorized, "unauthorized", "Authentication is required.", nil))
			return
		}
		if decision := s.access.Decide(context.Request.Context(), current, access.ActionInstall, record); !decision.Allowed {
			platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusForbidden, "private_skill_forbidden", "Access to this artifact is not allowed.", map[string]any{
				"packageName": packageName,
			}))
			return
		}
		if privateNext != nil {
			selected = privateNext
		}
		context.Header("Cache-Control", "private, no-store")
		selected.ServeHTTP(context.Writer, context.Request)
	}
}

func packageFromArtifactPath(escapedPath string) (string, bool) {
	path := strings.TrimPrefix(escapedPath, npmRegistryMountPath)
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return "", false
	}
	decoded, err := url.PathUnescape(path)
	if err != nil {
		return "", false
	}
	segments := strings.Split(decoded, "/")
	if len(segments) == 0 || segments[0] == "" {
		return "", false
	}
	if strings.HasPrefix(segments[0], "@") {
		if len(segments) < 2 {
			return "", false
		}
		if strings.Contains(segments[0], "%") || segments[0] == "@" || segments[1] == "" {
			return "", false
		}
		return segments[0] + "/" + segments[1], true
	}
	return segments[0], true
}
