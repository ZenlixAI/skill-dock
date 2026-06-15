package catalogapi

import (
	"net/http"

	routerpkg "skill_registry/app/catalogapi/router"
)

// Handler builds the HTTP routing tree with a gin engine while preserving the
// exported http.Handler contract.
func (s *Server) Handler() http.Handler {
	return routerpkg.New(s.routerConfig())
}
