package catalogapi

import (
	"net/http"

	"skill_registry/app/auth"
)

type SessionResponse struct {
	Role            string   `json:"role"`
	Actor           string   `json:"actor"`
	AuthMechanism   string   `json:"authMechanism"`
	AllowedActions  []string `json:"allowedActions"`
	AllowedPackages []string `json:"allowedPackages"`
}

func buildSessionResponse(role string, actor string, authMechanism string, allowedActions []string, allowedPackages []string) SessionResponse {
	packages := append([]string{}, allowedPackages...)

	return SessionResponse{
		Role:            role,
		Actor:           actor,
		AuthMechanism:   authMechanism,
		AllowedActions:  append([]string{}, allowedActions...),
		AllowedPackages: packages,
	}
}

func (s *Server) handleSession(writer http.ResponseWriter, request *http.Request) {
	current := auth.PrincipalFromContext(request.Context())
	writeJSON(writer, http.StatusOK, buildSessionResponse(
		current.Role,
		current.Actor,
		current.AuthMechanism,
		s.authorizer.AllowedActionsForRole(current.Role),
		current.AllowedPackages,
	))
}
