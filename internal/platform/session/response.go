package session

type Response struct {
	Role            string   `json:"role"`
	Actor           string   `json:"actor"`
	AuthMechanism   string   `json:"authMechanism"`
	AllowedActions  []string `json:"allowedActions"`
	AllowedPackages []string `json:"allowedPackages"`
}

func buildResponse(role string, actor string, authMechanism string, allowedActions []string, allowedPackages []string) Response {
	packages := append([]string{}, allowedPackages...)

	return Response{
		Role:            role,
		Actor:           actor,
		AuthMechanism:   authMechanism,
		AllowedActions:  append([]string{}, allowedActions...),
		AllowedPackages: packages,
	}
}
