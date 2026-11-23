package httpserver

import (
	"net/http"

	"github.com/gin-gonic/gin"

	catalog "skill_registry/internal/catalog/domain"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
)

func (s *Server) registerIdentityRoutes(engine *gin.Engine) {
	if engine == nil || s.identity == nil {
		return
	}
	engine.GET("/api/v1/me", s.meHandler)
	engine.GET("/api/v1/me/orgs", s.meOrganizationsHandler)
}

func (s *Server) meHandler(context *gin.Context) {
	current := auth.PrincipalFromContext(context.Request.Context())
	if !current.IsUser() {
		platformhttp.WriteCatalogError(context.Writer, context.Request, identityRequiredError())
		return
	}
	me, err := s.identity.Me(context.Request.Context(), current)
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, identityUnavailableError())
		return
	}
	platformhttp.WriteJSON(context.Writer, http.StatusOK, me)
}

func (s *Server) meOrganizationsHandler(context *gin.Context) {
	current := auth.PrincipalFromContext(context.Request.Context())
	if !current.IsUser() {
		platformhttp.WriteCatalogError(context.Writer, context.Request, identityRequiredError())
		return
	}
	me, err := s.identity.Me(context.Request.Context(), current)
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, identityUnavailableError())
		return
	}
	platformhttp.WriteJSON(context.Writer, http.StatusOK, gin.H{"items": me.Organizations})
}

func identityRequiredError() *catalog.Error {
	return catalog.NewError(http.StatusUnauthorized, "unauthorized", "Authentication is required.", nil)
}

func identityUnavailableError() *catalog.Error {
	return catalog.NewError(http.StatusServiceUnavailable, "identity_unavailable", "Identity service is unavailable.", nil)
}
