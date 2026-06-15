package middleware

import (
	stdhttp "net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"skill_registry/business/catalog"
	platformauth "skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
	platformobservability "skill_registry/internal/platform/observability"
)

type AuthenticateFunc func(request *stdhttp.Request) (platformauth.Principal, bool)
type AuthorizeActionFunc func(writer stdhttp.ResponseWriter, request *stdhttp.Request, action platformauth.Action, packageName string, version string) bool
type RecordAuthenticationDeniedFunc func(request *stdhttp.Request)

func WithAuthentication(authenticate AuthenticateFunc) gin.HandlerFunc {
	return func(context *gin.Context) {
		if authenticate == nil {
			context.Next()
			return
		}
		current, ok := authenticate(context.Request)
		if ok {
			context.Request = context.Request.WithContext(platformauth.ContextWithPrincipal(context.Request.Context(), current))
			platformhttp.SetObservedPrincipal(context.Request, current)
		}
		context.Next()
	}
}

func RequireAction(next gin.HandlerFunc, action platformauth.Action, authorize AuthorizeActionFunc, recordDenied RecordAuthenticationDeniedFunc) gin.HandlerFunc {
	return func(context *gin.Context) {
		platformhttp.SetObservedAction(context.Request, string(action))

		current := platformauth.PrincipalFromContext(context.Request.Context())
		if !principalPresent(current) {
			platformhttp.SetObservedAuthDecision(context.Request, platformobservability.AuthDecisionDenied)
			platformhttp.SetObservedOutcome(context.Request, "unauthorized")
			platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(stdhttp.StatusUnauthorized, "unauthorized", "Authentication is required.", nil))
			if recordDenied != nil {
				recordDenied(context.Request)
			}
			context.Abort()
			return
		}

		if authorize != nil && !authorize(context.Writer, context.Request, action, "", "") {
			context.Abort()
			return
		}

		next(context)
	}
}

func principalPresent(current platformauth.Principal) bool {
	return strings.TrimSpace(current.Role) != "" ||
		strings.TrimSpace(current.Actor) != "" ||
		strings.TrimSpace(current.AuthMechanism) != "" ||
		len(current.AllowedPackages) > 0
}
