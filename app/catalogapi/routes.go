package catalogapi

import (
	"net/http"

	"skill_registry/app/auth"
	routerpkg "skill_registry/app/catalogapi/router"
)

// Handler builds the HTTP routing tree with a gin engine while preserving the
// exported http.Handler contract.
func (s *Server) Handler() http.Handler {
	return routerpkg.New(s.routerConfig())
}

// The skill subtree adapters resolve the {package} and {version} path values,
// enforce package-scoped access (and the action-specific authorization that
// mutating routes require) before delegating to the concrete handler.

func (s *Server) handleSkillGovernanceRoute(writer http.ResponseWriter, request *http.Request) {
	packageName := request.PathValue("package")
	if !s.authorizePackageAccess(writer, request, packageName) {
		return
	}
	if !s.authorizeActionAccess(writer, request, auth.ActionApplySkillGovernance, packageName, "") {
		return
	}
	s.handleSkillGovernanceChange(writer, request, packageName)
}

func (s *Server) handleReleaseRecordRoute(writer http.ResponseWriter, request *http.Request) {
	packageName := request.PathValue("package")
	version := request.PathValue("version")
	if !s.authorizePackageAccess(writer, request, packageName) {
		return
	}
	if !s.authorizeActionAccess(writer, request, auth.ActionCreateReleaseRecord, packageName, version) {
		return
	}
	s.handleReleaseRecord(writer, request, packageName, version)
}

func (s *Server) handleVersionGovernanceRoute(writer http.ResponseWriter, request *http.Request) {
	packageName := request.PathValue("package")
	version := request.PathValue("version")
	if !s.authorizePackageAccess(writer, request, packageName) {
		return
	}
	if !s.authorizeActionAccess(writer, request, auth.ActionApplyVersionGovernance, packageName, version) {
		return
	}
	s.handleVersionGovernanceChange(writer, request, packageName, version)
}
