package httpserver

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"skill_registry/internal/access"
	catalog "skill_registry/internal/catalog/domain"
	"skill_registry/internal/identity"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
)

func (s *Server) registerOwnershipRoutes(engine *gin.Engine) {
	if engine == nil || s.accessStore == nil {
		return
	}
	engine.GET("/api/v1/orgs/:org/namespaces", s.listNamespacesHandler)
	engine.POST("/api/v1/orgs/:org/namespaces", s.createNamespaceHandler)
	engine.POST("/api/v1/orgs/:org/namespaces/:id/verify", s.verifyNamespaceHandler)
	engine.GET("/api/v1/orgs/:org/artifact-config", s.getArtifactConfigHandler)
	engine.PUT("/api/v1/orgs/:org/artifact-config", s.putArtifactConfigHandler)
	engine.PATCH("/api/v1/skills/:package/visibility", s.visibilityHandler)
	engine.GET("/api/v1/skills/:package/grants", s.listGrantsHandler)
	engine.POST("/api/v1/skills/:package/grants", s.createGrantHandler)
	engine.DELETE("/api/v1/skills/:package/grants/:id", s.deleteGrantHandler)
	engine.GET("/api/v1/skills/:package/maintainers", s.listMaintainersHandler)
	engine.POST("/api/v1/skills/:package/maintainers", s.addMaintainerHandler)
	engine.DELETE("/api/v1/skills/:package/maintainers/:subject", s.removeMaintainerHandler)
}

func (s *Server) listNamespacesHandler(context *gin.Context) {
	organization, err := s.resolveTenant(context, context.Param("org"))
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	items, storeErr := s.accessStore.ListNamespaces(context.Request.Context(), organization.ID)
	if storeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	platformhttp.WriteJSON(context.Writer, http.StatusOK, gin.H{"items": items})
}

func (s *Server) createNamespaceHandler(context *gin.Context) {
	organization, err := s.resolveTenant(context, context.Param("org"))
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	if !s.canManageTenant(context, organization.ID) {
		platformhttp.WriteCatalogError(context.Writer, context.Request, forbiddenError("org:manage is required"))
		return
	}
	var payload struct {
		Prefix string `json:"prefix"`
	}
	if err := platformhttp.DecodeJSONBody(context.Request, &payload); err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	namespace, storeErr := s.accessStore.PutNamespace(context.Request.Context(), access.Namespace{
		Prefix:         payload.Prefix,
		OrganizationID: organization.ID,
	})
	if storeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusConflict, "namespace_conflict", "Namespace could not be created.", map[string]any{"prefix": payload.Prefix}))
		return
	}
	s.readCache.InvalidateAll()
	platformhttp.WriteJSON(context.Writer, http.StatusCreated, namespace)
}

func (s *Server) verifyNamespaceHandler(context *gin.Context) {
	organization, err := s.resolveTenant(context, context.Param("org"))
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	current := auth.PrincipalFromContext(context.Request.Context())
	if !current.PlatformAdmin {
		platformhttp.WriteCatalogError(context.Writer, context.Request, forbiddenError("platform administrator is required"))
		return
	}
	namespace, storeErr := s.accessStore.VerifyNamespace(context.Request.Context(), organization.ID, context.Param("id"))
	if storeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusNotFound, "namespace_not_found", "Namespace not found.", nil))
		return
	}
	s.readCache.InvalidateAll()
	platformhttp.WriteJSON(context.Writer, http.StatusOK, namespace)
}

func (s *Server) getArtifactConfigHandler(context *gin.Context) {
	organization, err := s.resolveTenant(context, context.Param("org"))
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	config, storeErr := s.accessStore.GetArtifactConfig(context.Request.Context(), organization.ID)
	if storeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusNotFound, "artifact_config_not_found", "Private artifact target is not configured.", nil))
		return
	}
	platformhttp.WriteJSON(context.Writer, http.StatusOK, gin.H{
		"organizationId": config.OrganizationID,
		"scopePrefix":    config.ScopePrefix,
		"proxyMountPath": config.ProxyMountPath,
		"authMode":       config.AuthMode,
	})
}

func (s *Server) putArtifactConfigHandler(context *gin.Context) {
	organization, err := s.resolveTenant(context, context.Param("org"))
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	if !s.canManageTenant(context, organization.ID) {
		platformhttp.WriteCatalogError(context.Writer, context.Request, forbiddenError("org:manage is required"))
		return
	}
	var payload struct {
		UpstreamURL    string `json:"upstreamUrl"`
		ScopePrefix    string `json:"scopePrefix"`
		ProxyMountPath string `json:"proxyMountPath"`
	}
	if decodeErr := platformhttp.DecodeJSONBody(context.Request, &payload); decodeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, decodeErr)
		return
	}
	config, storeErr := s.accessStore.PutArtifactConfig(context.Request.Context(), access.ArtifactConfig{
		OrganizationID: organization.ID,
		UpstreamURL:    payload.UpstreamURL,
		ScopePrefix:    payload.ScopePrefix,
		ProxyMountPath: payload.ProxyMountPath,
		AuthMode:       "catalog_proxy",
	})
	if storeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "Artifact target could not be saved.", map[string]any{"reason": storeErr.Error()}))
		return
	}
	platformhttp.WriteJSON(context.Writer, http.StatusOK, gin.H{
		"organizationId": config.OrganizationID,
		"scopePrefix":    config.ScopePrefix,
		"proxyMountPath": config.ProxyMountPath,
		"authMode":       config.AuthMode,
	})
}

func (s *Server) visibilityHandler(context *gin.Context) {
	record, organization, err := s.loadManagedSkill(context, context.Param("package"), access.ActionManage)
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	var payload struct {
		Visibility string `json:"visibility"`
	}
	if decodeErr := platformhttp.DecodeJSONBody(context.Request, &payload); decodeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, decodeErr)
		return
	}
	visibility := strings.ToLower(strings.TrimSpace(payload.Visibility))
	if visibility != "public" && visibility != "organization" && visibility != "private" {
		platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "visibility must be public, organization, or private.", nil))
		return
	}
	if visibility == "public" {
		if record.OwnerType != "organization" || record.OrganizationID == "" {
			platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusConflict, "namespace_forbidden", "Public skills require an organization owner.", nil))
			return
		}
		namespace, namespaceErr := s.accessStore.GetNamespaceForPackage(context.Request.Context(), record.PackageName)
		if namespaceErr != nil || namespace.Status != "verified" {
			platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusConflict, "namespace_unverified", "Namespace is not verified for public publishing.", nil))
			return
		}
	}
	record.Visibility = visibility
	if storeErr := s.accessStore.PutSkillAccess(context.Request.Context(), record); storeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	s.readCache.InvalidateAll()
	platformhttp.WriteJSON(context.Writer, http.StatusOK, gin.H{"packageName": record.PackageName, "visibility": visibility, "organizationId": organization.ID})
}

func (s *Server) listGrantsHandler(context *gin.Context) {
	record, organization, err := s.loadManagedSkill(context, context.Param("package"), access.ActionManage)
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	items, storeErr := s.accessStore.ListGrants(context.Request.Context(), organization.ID, record.PackageName)
	if storeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	platformhttp.WriteJSON(context.Writer, http.StatusOK, gin.H{"items": items})
}

func (s *Server) createGrantHandler(context *gin.Context) {
	record, organization, err := s.loadManagedSkill(context, context.Param("package"), access.ActionGrant)
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	var payload struct {
		SubjectType string `json:"subjectType"`
		SubjectID   string `json:"subjectId"`
		Permission  string `json:"permission"`
		ExpiresAt   string `json:"expiresAt,omitempty"`
	}
	if decodeErr := platformhttp.DecodeJSONBody(context.Request, &payload); decodeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, decodeErr)
		return
	}
	if payload.SubjectType != "user" {
		platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "Only user grants are supported in v1.", nil))
		return
	}
	if payload.Permission != "view" && payload.Permission != "install" && payload.Permission != "publish" && payload.Permission != "manage" {
		platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "permission is invalid.", nil))
		return
	}
	if s.identityStore != nil {
		member, memberErr := s.identityStore.IsMember(context.Request.Context(), organization.ID, payload.SubjectID)
		if memberErr != nil || !member {
			platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusUnprocessableEntity, "cross_tenant_forbidden", "Grant subject must be a member of the owning tenant.", nil))
			return
		}
	}
	grant := access.Grant{
		OrganizationID: organization.ID,
		PackageName:    record.PackageName,
		SubjectType:    payload.SubjectType,
		SubjectID:      payload.SubjectID,
		Permission:     payload.Permission,
		GrantedBy:      auth.PrincipalFromContext(context.Request.Context()).UserID,
	}
	if payload.ExpiresAt != "" {
		expiresAt, parseErr := time.Parse(time.RFC3339, payload.ExpiresAt)
		if parseErr != nil {
			platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "expiresAt must be RFC3339.", nil))
			return
		}
		grant.ExpiresAt = &expiresAt
	}
	if storeErr := s.accessStore.PutGrant(context.Request.Context(), grant); storeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	s.readCache.InvalidateAll()
	platformhttp.WriteJSON(context.Writer, http.StatusCreated, grant)
}

func (s *Server) deleteGrantHandler(context *gin.Context) {
	_, organization, err := s.loadManagedSkill(context, context.Param("package"), access.ActionGrant)
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	if storeErr := s.accessStore.DeleteGrant(context.Request.Context(), organization.ID, context.Param("package"), context.Param("id")); storeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	s.readCache.InvalidateAll()
	context.Status(http.StatusNoContent)
}

func (s *Server) listMaintainersHandler(context *gin.Context) {
	record, _, err := s.loadManagedSkill(context, context.Param("package"), access.ActionRead)
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	users, storeErr := s.accessStore.ListMaintainers(context.Request.Context(), record.PackageName)
	if storeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	platformhttp.WriteJSON(context.Writer, http.StatusOK, gin.H{"items": users})
}

func (s *Server) addMaintainerHandler(context *gin.Context) {
	record, organization, err := s.loadManagedSkill(context, context.Param("package"), access.ActionManage)
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	var payload struct {
		UserID string `json:"userId"`
	}
	if decodeErr := platformhttp.DecodeJSONBody(context.Request, &payload); decodeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, decodeErr)
		return
	}
	if s.identityStore != nil {
		member, memberErr := s.identityStore.IsMember(context.Request.Context(), organization.ID, payload.UserID)
		if memberErr != nil || !member {
			platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusUnprocessableEntity, "cross_tenant_forbidden", "Maintainer must be a member of the owning tenant.", nil))
			return
		}
	}
	if storeErr := s.accessStore.AddMaintainer(context.Request.Context(), record.PackageName, organization.ID, payload.UserID, auth.PrincipalFromContext(context.Request.Context()).UserID); storeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	s.readCache.InvalidateAll()
	context.Status(http.StatusNoContent)
}

func (s *Server) removeMaintainerHandler(context *gin.Context) {
	record, _, err := s.loadManagedSkill(context, context.Param("package"), access.ActionManage)
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	if storeErr := s.accessStore.RemoveMaintainer(context.Request.Context(), record.PackageName, context.Param("subject")); storeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	s.readCache.InvalidateAll()
	context.Status(http.StatusNoContent)
}

func (s *Server) loadManagedSkill(context *gin.Context, packageName string, action access.Action) (access.SkillAccess, identity.Organization, *catalog.Error) {
	if s == nil || s.accessStore == nil || s.access == nil {
		return access.SkillAccess{}, identity.Organization{}, catalog.NewError(http.StatusServiceUnavailable, "feature_disabled", "Skill access control is not enabled.", nil)
	}
	record, err := s.accessStore.GetSkillAccess(context.Request.Context(), packageName)
	if err != nil {
		return access.SkillAccess{}, identity.Organization{}, catalog.NewError(http.StatusNotFound, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}
	if record.OrganizationID == "" {
		return record, identity.Organization{}, forbiddenError("Skill is not owned by a tenant.")
	}
	if s.identityStore == nil {
		return record, identity.Organization{}, catalog.NewError(http.StatusServiceUnavailable, "identity_unavailable", "Identity service is unavailable.", nil)
	}
	organization, orgErr := s.identityStore.GetOrganizationByID(context.Request.Context(), record.OrganizationID)
	if orgErr != nil {
		return access.SkillAccess{}, identity.Organization{}, internalError()
	}
	current := auth.PrincipalFromContext(context.Request.Context())
	if !current.PlatformAdmin && current.TenantID != organization.ID {
		return access.SkillAccess{}, identity.Organization{}, catalog.NewError(http.StatusNotFound, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}
	if decision := s.access.Decide(context.Request.Context(), current, action, record); !decision.Allowed {
		return access.SkillAccess{}, identity.Organization{}, forbiddenError(decision.Reason)
	}
	return record, organization, nil
}

func (s *Server) resolveTenant(context *gin.Context, value string) (identity.Organization, *catalog.Error) {
	if s.identityStore == nil {
		return identity.Organization{}, catalog.NewError(http.StatusServiceUnavailable, "identity_unavailable", "Identity service is unavailable.", nil)
	}
	current := auth.PrincipalFromContext(context.Request.Context())
	if !current.IsUser() && !current.PlatformAdmin && !current.IsService() {
		return identity.Organization{}, catalog.NewError(http.StatusUnauthorized, "unauthorized", "Authentication is required.", nil)
	}
	organization, err := s.identityStore.GetOrganizationBySlug(context.Request.Context(), value)
	if err != nil {
		organization, err = s.identityStore.GetOrganizationByID(context.Request.Context(), value)
	}
	if err != nil {
		return identity.Organization{}, catalog.NewError(http.StatusNotFound, "tenant_not_found", "Organization not found.", nil)
	}
	if current.PlatformAdmin {
		return organization, nil
	}
	if current.TenantID != organization.ID && current.TenantSlug != organization.Slug {
		return identity.Organization{}, catalog.NewError(http.StatusForbidden, "tenant_mismatch", "Organization does not match the active tenant.", nil)
	}
	return organization, nil
}

func (s *Server) canManageTenant(context *gin.Context, organizationID string) bool {
	current := auth.PrincipalFromContext(context.Request.Context())
	if current.PlatformAdmin {
		return true
	}
	if current.TenantID != organizationID {
		return false
	}
	return current.HasScope("org:manage")
}

func forbiddenError(reason string) *catalog.Error {
	return catalog.NewError(http.StatusForbidden, "forbidden", "Action is not allowed.", map[string]any{"reason": reason})
}

func internalError() *catalog.Error {
	return catalog.NewError(http.StatusInternalServerError, "internal_error", "Internal server error.", nil)
}
