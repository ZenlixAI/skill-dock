package catalogapi

import (
	"net/http"

	"skill_registry/app/auth"
)

// Handler builds the HTTP routing tree. Path matching relies on the standard
// library ServeMux (Go 1.22+ method + wildcard patterns): the encoded package
// segment (e.g. %40scope%2Fname) is matched as a single path element and
// exposed already-decoded through Request.PathValue.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/healthz", http.HandlerFunc(s.handleHealthz))
	mux.Handle("/readyz", http.HandlerFunc(s.handleReadyz))
	mux.Handle("/metrics", http.HandlerFunc(s.handleMetrics))
	mux.Handle("/api/v1/session", http.HandlerFunc(s.handleSession))
	mux.Handle("/api/v1/session/login", http.HandlerFunc(s.handleSessionLogin))
	mux.Handle("/api/v1/session/logout", http.HandlerFunc(s.handleSessionLogout))

	if s.mode.servesWriteRoutes() {
		mux.Handle("GET /api/v1/audit-events", s.requireAction(s.handleAuditEvents, auth.ActionReadAuditEvents))
	}
	if s.mode.servesReadRoutes() {
		s.registerCatalogReadRoutes(mux)
	}
	if s.mode.servesWriteRoutes() {
		s.registerCatalogWriteRoutes(mux)
		s.registerPackagingRoutes(mux)
	}
	if !s.disableMockRegistry {
		mux.Handle("/mock-npm/", http.HandlerFunc(s.handleMockRegistry))
	}

	return s.withCORS(s.withRequestMetadata(s.withObservability(mux)))
}

func (s *Server) registerCatalogReadRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/skills", s.requireAction(s.handleSkills, auth.ActionReadCatalog))
	mux.Handle("GET /api/v1/skills/search", s.requireAction(s.handleSkills, auth.ActionReadCatalog))
	mux.Handle("GET /api/v1/skills/curated", s.requireAction(s.handleCuratedSkills, auth.ActionReadCatalog))
	mux.Handle("GET /api/v1/governance-changes", s.requireAction(s.handleGovernanceChanges, auth.ActionReadCatalog))
	mux.Handle("GET /api/v1/skills/{package}", s.requireAction(s.handleSkillDetailRoute, auth.ActionReadCatalog))
	mux.Handle("GET /api/v1/skills/{package}/versions", s.requireAction(s.handleVersionListRoute, auth.ActionReadCatalog))
	mux.Handle("GET /api/v1/skills/{package}/versions/{version}", s.requireAction(s.handleVersionDetailRoute, auth.ActionReadCatalog))
	mux.Handle("GET /api/v1/skills/{package}/install-target", s.requireAction(s.handleInstallTargetRoute, auth.ActionReadCatalog))
}

func (s *Server) registerCatalogWriteRoutes(mux *http.ServeMux) {
	mux.Handle("POST /api/v1/skills/{package}/governance-changes", s.requireAction(s.handleSkillGovernanceRoute, auth.ActionReadCatalog))
	mux.Handle("PUT /api/v1/skills/{package}/versions/{version}/release-record", s.requireAction(s.handleReleaseRecordRoute, auth.ActionReadCatalog))
	mux.Handle("POST /api/v1/skills/{package}/versions/{version}/governance-changes", s.requireAction(s.handleVersionGovernanceRoute, auth.ActionReadCatalog))
}

func (s *Server) registerPackagingRoutes(mux *http.ServeMux) {
	mux.Handle("/api/v1/skill-packaging/prepare", s.requireAction(s.handlePackagingPrepare, auth.ActionPackagingPrepare))
	mux.Handle("/api/v1/skill-packaging/validate", s.requireAction(s.handlePackagingValidate, auth.ActionPackagingValidate))
	mux.Handle("/api/v1/skill-packaging/pack", s.requireAction(s.handlePackagingPack, auth.ActionPackagingPack))
	mux.Handle("/api/v1/skill-packaging/publish", s.requireAction(s.handlePackagingPublish, auth.ActionPackagingPublish))
	mux.Handle("/api/v1/skill-packaging/cleanup", s.requireAction(s.handlePackagingCleanup, auth.ActionPackagingPack))
	mux.Handle("/api/v1/skill-packaging/bundle", s.requireAction(s.handlePackagingBundleDownload, auth.ActionPackagingPack))
}

// The skill subtree adapters resolve the {package} and {version} path values,
// enforce package-scoped access (and the action-specific authorization that
// mutating routes require) before delegating to the concrete handler.

func (s *Server) handleSkillDetailRoute(writer http.ResponseWriter, request *http.Request) {
	packageName := request.PathValue("package")
	if !s.authorizePackageAccess(writer, request, packageName) {
		return
	}
	s.handleSkillDetail(writer, request, packageName)
}

func (s *Server) handleVersionListRoute(writer http.ResponseWriter, request *http.Request) {
	packageName := request.PathValue("package")
	if !s.authorizePackageAccess(writer, request, packageName) {
		return
	}
	s.handleVersionList(writer, request, packageName)
}

func (s *Server) handleVersionDetailRoute(writer http.ResponseWriter, request *http.Request) {
	packageName := request.PathValue("package")
	if !s.authorizePackageAccess(writer, request, packageName) {
		return
	}
	s.handleVersionDetail(writer, request, packageName, request.PathValue("version"))
}

func (s *Server) handleInstallTargetRoute(writer http.ResponseWriter, request *http.Request) {
	packageName := request.PathValue("package")
	if !s.authorizePackageAccess(writer, request, packageName) {
		return
	}
	s.handleInstallTarget(writer, request, packageName)
}

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
