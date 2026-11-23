package httpserver

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"skill_registry/internal/access"
	catalog "skill_registry/internal/catalog/domain"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
	"skill_registry/internal/tags"
)

func (s *Server) registerTagRoutes(engine *gin.Engine) {
	if engine == nil || s.tagStore == nil {
		return
	}
	engine.GET("/api/v1/tags", s.listTagsHandler)
	engine.POST("/api/v1/tags", s.createTagHandler)
	engine.PATCH("/api/v1/tags/:id", s.updateTagHandler)
	engine.DELETE("/api/v1/tags/:id", s.deleteTagHandler)
	engine.PUT("/api/v1/skills/:package/tags", s.replaceSkillTagsHandler)
}

func (s *Server) listTagsHandler(context *gin.Context) {
	current := auth.PrincipalFromContext(context.Request.Context())
	items, err := s.tagStore.ListVisible(context.Request.Context(), current.TenantID)
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	platformhttp.WriteJSON(context.Writer, http.StatusOK, gin.H{"items": items})
}

func (s *Server) createTagHandler(context *gin.Context) {
	current := auth.PrincipalFromContext(context.Request.Context())
	var payload struct {
		Slug       string `json:"slug"`
		Label      string `json:"label"`
		Scope      string `json:"scope"`
		Visibility string `json:"visibility"`
	}
	if err := platformhttp.DecodeJSONBody(context.Request, &payload); err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	scope := strings.ToLower(strings.TrimSpace(payload.Scope))
	if scope == "" {
		scope = "tenant"
	}
	organizationID := current.TenantID
	if scope == "system" {
		if !current.PlatformAdmin {
			platformhttp.WriteCatalogError(context.Writer, context.Request, forbiddenError("platform administrator is required for system tags"))
			return
		}
		organizationID = ""
	} else if scope != "tenant" || current.TenantID == "" || !(current.HasScope("org:manage") || current.PlatformAdmin) {
		platformhttp.WriteCatalogError(context.Writer, context.Request, forbiddenError("org:manage is required"))
		return
	}
	tag, err := s.tagStore.Create(context.Request.Context(), tags.Tag{
		Slug:           payload.Slug,
		Label:          payload.Label,
		Scope:          scope,
		OrganizationID: organizationID,
		Visibility:     payload.Visibility,
		CreatedBy:      current.UserID,
	})
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "Tag could not be created.", map[string]any{"reason": err.Error()}))
		return
	}
	s.readCache.InvalidateAll()
	platformhttp.WriteJSON(context.Writer, http.StatusCreated, tag)
}

func (s *Server) updateTagHandler(context *gin.Context) {
	current := auth.PrincipalFromContext(context.Request.Context())
	if current.TenantID == "" || !(current.HasScope("org:manage") || current.PlatformAdmin) {
		platformhttp.WriteCatalogError(context.Writer, context.Request, forbiddenError("org:manage is required"))
		return
	}
	tag, err := s.tagStore.Get(context.Request.Context(), context.Param("id"))
	if err != nil || (tag.Scope == "tenant" && tag.OrganizationID != current.TenantID) {
		platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusNotFound, "tag_not_found", "Tag not found.", nil))
		return
	}
	var payload struct {
		Label      string `json:"label"`
		Visibility string `json:"visibility"`
	}
	if decodeErr := platformhttp.DecodeJSONBody(context.Request, &payload); decodeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, decodeErr)
		return
	}
	tag.Label = payload.Label
	tag.Visibility = payload.Visibility
	updated, updateErr := s.tagStore.Update(context.Request.Context(), tag)
	if updateErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	s.readCache.InvalidateAll()
	platformhttp.WriteJSON(context.Writer, http.StatusOK, updated)
}

func (s *Server) deleteTagHandler(context *gin.Context) {
	current := auth.PrincipalFromContext(context.Request.Context())
	if current.TenantID == "" || !(current.HasScope("org:manage") || current.PlatformAdmin) {
		platformhttp.WriteCatalogError(context.Writer, context.Request, forbiddenError("org:manage is required"))
		return
	}
	tag, err := s.tagStore.Get(context.Request.Context(), context.Param("id"))
	if err != nil || (tag.Scope == "tenant" && tag.OrganizationID != current.TenantID) {
		platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusNotFound, "tag_not_found", "Tag not found.", nil))
		return
	}
	if err := s.tagStore.Delete(context.Request.Context(), tag.ID); err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	s.readCache.InvalidateAll()
	context.Status(http.StatusNoContent)
}

func (s *Server) replaceSkillTagsHandler(context *gin.Context) {
	record, organization, err := s.loadManagedSkill(context, context.Param("package"), access.ActionManage)
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	var payload struct {
		TagIDs []string `json:"tagIds"`
	}
	if decodeErr := platformhttp.DecodeJSONBody(context.Request, &payload); decodeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, decodeErr)
		return
	}
	visible, listErr := s.tagStore.ListVisible(context.Request.Context(), organization.ID)
	if listErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	allowed := map[string]struct{}{}
	for _, tag := range visible {
		allowed[tag.ID] = struct{}{}
	}
	for _, id := range payload.TagIDs {
		if _, ok := allowed[id]; !ok {
			platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "Tag is not visible to the active tenant.", map[string]any{"tagId": id}))
			return
		}
	}
	if err := s.tagStore.ReplaceSkillTags(context.Request.Context(), organization.ID, record.PackageName, payload.TagIDs, auth.PrincipalFromContext(context.Request.Context()).UserID); err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	s.readCache.InvalidateAll()
	context.Status(http.StatusNoContent)
}

func (s *Server) tagsForPrincipal(ctx context.Context, packageName string, current auth.Principal) []catalog.TagSummary {
	if s == nil || s.tagStore == nil {
		return nil
	}
	items, err := s.tagStore.ListForPackage(ctx, packageName)
	if err != nil {
		return nil
	}
	return filterTags(items, current)
}

func (s *Server) enrichSummaryTags(ctx context.Context, current auth.Principal, items []catalog.SkillSummary) {
	if s == nil || s.tagStore == nil || len(items) == 0 {
		return
	}
	packageNames := make([]string, 0, len(items))
	for _, item := range items {
		packageNames = append(packageNames, item.PackageName)
	}
	byPackage, err := s.tagStore.ListForPackages(ctx, packageNames)
	if err != nil {
		return
	}
	for index := range items {
		items[index].Tags = filterTags(byPackage[items[index].PackageName], current)
	}
}

func filterTags(items []tags.Tag, current auth.Principal) []catalog.TagSummary {
	result := make([]catalog.TagSummary, 0, len(items))
	for _, tag := range items {
		if tag.Scope == "tenant" && tag.OrganizationID != current.TenantID {
			continue
		}
		if tag.Visibility == "private" && current.TenantID == "" && !current.PlatformAdmin {
			continue
		}
		result = append(result, catalog.TagSummary{Slug: tag.Slug, Label: tag.Label, Scope: tag.Scope})
	}
	return result
}
