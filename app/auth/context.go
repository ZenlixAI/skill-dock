package auth

import (
	"context"
	"net/http"
)

type principalContextKeyType string

const principalContextKey principalContextKeyType = "principal"

func ContextWithPrincipal(ctx context.Context, current Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, current)
}

func PrincipalFromContext(ctx context.Context) Principal {
	current, _ := ctx.Value(principalContextKey).(Principal)
	return current
}

func RoleFromContext(ctx context.Context) string {
	return PrincipalFromContext(ctx).Role
}

func ActorFromRequest(request *http.Request) string {
	return PrincipalFromContext(request.Context()).Actor
}
