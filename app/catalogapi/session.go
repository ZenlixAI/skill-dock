package catalogapi

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
