package auth

import platformauth "skill_registry/internal/platform/auth"

var (
	ContextWithPrincipal = platformauth.ContextWithPrincipal
	PrincipalFromContext = platformauth.PrincipalFromContext
	RoleFromContext      = platformauth.RoleFromContext
	ActorFromRequest     = platformauth.ActorFromRequest
)
