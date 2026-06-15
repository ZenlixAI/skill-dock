package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/common"
	"skill_registry/app/observability"
	"skill_registry/business/catalog"
)

type AuthenticateFunc func(request *http.Request) (auth.Principal, bool)
type AuthorizeActionFunc func(writer http.ResponseWriter, request *http.Request, action auth.Action, packageName string, version string) bool
type RecordAuthenticationDeniedFunc func(request *http.Request)

func WithAuthentication(authenticate AuthenticateFunc) gin.HandlerFunc {
	return func(context *gin.Context) {
		if authenticate == nil {
			context.Next()
			return
		}
		current, ok := authenticate(context.Request)
		if ok {
			context.Request = context.Request.WithContext(auth.ContextWithPrincipal(context.Request.Context(), current))
			SetObservedPrincipal(context.Request, current)
		}
		context.Next()
	}
}

func RequireAction(next gin.HandlerFunc, action auth.Action, authorize AuthorizeActionFunc, recordDenied RecordAuthenticationDeniedFunc) gin.HandlerFunc {
	return func(context *gin.Context) {
		SetObservedAction(context.Request, string(action))

		current := auth.PrincipalFromContext(context.Request.Context())
		if !principalPresent(current) {
			SetObservedAuthDecision(context.Request, observability.AuthDecisionDenied)
			SetObservedOutcome(context.Request, "unauthorized")
			common.WriteCatalogError(context.Writer, context.Request, catalog.NewError(401, "unauthorized", "Authentication is required.", nil))
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

func principalPresent(current auth.Principal) bool {
	return strings.TrimSpace(current.Role) != "" ||
		strings.TrimSpace(current.Actor) != "" ||
		strings.TrimSpace(current.AuthMechanism) != "" ||
		len(current.AllowedPackages) > 0
}
