package middleware

import platformmiddleware "skill_registry/internal/platform/http/middleware"

type AuthenticateFunc = platformmiddleware.AuthenticateFunc
type AuthorizeActionFunc = platformmiddleware.AuthorizeActionFunc
type RecordAuthenticationDeniedFunc = platformmiddleware.RecordAuthenticationDeniedFunc

var (
	WithAuthentication = platformmiddleware.WithAuthentication
	RequireAction      = platformmiddleware.RequireAction
)
