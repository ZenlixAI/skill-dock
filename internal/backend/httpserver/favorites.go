package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"skill_registry/internal/access"
	catalog "skill_registry/internal/catalog/domain"
	"skill_registry/internal/favorites"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
)

type favoriteRef struct {
	RefKey           string `json:"refKey,omitempty"`
	RefType          string `json:"refType"`
	PackageName      string `json:"packageName,omitempty"`
	ExternalProvider string `json:"externalProvider,omitempty"`
	ExternalSource   string `json:"externalSource,omitempty"`
	ExternalSkillID  string `json:"externalSkillId,omitempty"`
}

func (s *Server) registerFavoriteRoutes(engine *gin.Engine) {
	if engine == nil || s.favoriteStore == nil {
		return
	}
	engine.GET("/api/v1/me/favorites", s.listFavoritesHandler)
	engine.POST("/api/v1/me/favorites", s.upsertFavoriteHandler)
	engine.DELETE("/api/v1/me/favorites", s.deleteFavoriteHandler)
	engine.GET("/api/v1/me/favorites/status", s.favoriteStatusHandler)
	engine.POST("/api/v1/me/favorites/merge", s.mergeFavoritesHandler)
}

func (s *Server) listFavoritesHandler(context *gin.Context) {
	current := requireFavoriteUser(context)
	if current.UserID == "" {
		return
	}
	items, err := s.favoriteStore.List(context.Request.Context(), current.UserID, current.TenantID, current.TenantID == "", 100, 0)
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	items = s.markFavoriteAvailability(context.Request.Context(), current, items)
	platformhttp.WriteJSON(context.Writer, http.StatusOK, gin.H{"items": items})
}

func (s *Server) upsertFavoriteHandler(context *gin.Context) {
	current := requireFavoriteUser(context)
	if current.UserID == "" {
		return
	}
	var payload struct {
		Ref      favoriteRef     `json:"ref"`
		Snapshot json.RawMessage `json:"snapshot,omitempty"`
	}
	if err := platformhttp.DecodeJSONBody(context.Request, &payload); err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	favorite, err := s.buildFavorite(context.Request.Context(), current, payload.Ref, payload.Snapshot)
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	saved, storeErr := s.favoriteStore.Upsert(context.Request.Context(), favorite)
	if storeErr != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	platformhttp.WriteJSON(context.Writer, http.StatusCreated, saved)
}

func (s *Server) deleteFavoriteHandler(context *gin.Context) {
	current := requireFavoriteUser(context)
	if current.UserID == "" {
		return
	}
	refKey := strings.TrimSpace(context.Query("refKey"))
	if refKey == "" {
		platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "refKey is required.", nil))
		return
	}
	if err := s.favoriteStore.Delete(context.Request.Context(), current.UserID, refKey); err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	context.Status(http.StatusNoContent)
}

func (s *Server) favoriteStatusHandler(context *gin.Context) {
	current := requireFavoriteUser(context)
	if current.UserID == "" {
		return
	}
	status, err := s.favoriteStore.Status(context.Request.Context(), current.UserID, context.QueryArray("refKey"))
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	platformhttp.WriteJSON(context.Writer, http.StatusOK, gin.H{"status": status})
}

func (s *Server) mergeFavoritesHandler(context *gin.Context) {
	current := requireFavoriteUser(context)
	if current.UserID == "" {
		return
	}
	var payload struct {
		Favorites []struct {
			Ref      favoriteRef     `json:"ref"`
			Snapshot json.RawMessage `json:"snapshot,omitempty"`
		} `json:"favorites"`
	}
	if err := platformhttp.DecodeJSONBody(context.Request, &payload); err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}
	for _, item := range payload.Favorites {
		favorite, err := s.buildFavorite(context.Request.Context(), current, item.Ref, item.Snapshot)
		if err != nil {
			continue
		}
		_, _ = s.favoriteStore.Upsert(context.Request.Context(), favorite)
	}
	items, err := s.favoriteStore.List(context.Request.Context(), current.UserID, current.TenantID, current.TenantID == "", 100, 0)
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, internalError())
		return
	}
	items = s.markFavoriteAvailability(context.Request.Context(), current, items)
	platformhttp.WriteJSON(context.Writer, http.StatusOK, gin.H{"items": items})
}

func (s *Server) buildFavorite(ctx context.Context, current auth.Principal, ref favoriteRef, snapshot json.RawMessage) (favorites.Favorite, error) {
	refType := strings.ToLower(strings.TrimSpace(ref.RefType))
	refKey := strings.TrimSpace(ref.RefKey)
	contextType := "global"
	organizationID := ""
	packageName := strings.TrimSpace(ref.PackageName)
	switch refType {
	case "catalog":
		if packageName == "" {
			return favorites.Favorite{}, catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "packageName is required for catalog favorites.", nil)
		}
		refKey = "catalog:" + packageName
		if s.accessStore != nil {
			record, err := s.accessStore.GetSkillAccess(ctx, packageName)
			if err == nil {
				switch {
				case record.OwnerType == "user" && record.OwnerID == current.UserID:
					contextType = "personal"
				case record.Visibility == "public" || record.Visibility == "":
					contextType = "global"
				default:
					if current.TenantID != record.OrganizationID {
						return favorites.Favorite{}, catalog.NewError(http.StatusForbidden, "tenant_mismatch", "Active tenant does not match the skill tenant.", nil)
					}
					contextType = "tenant"
					organizationID = record.OrganizationID
				}
			}
		}
	case "external":
		if ref.ExternalProvider == "" || ref.ExternalSource == "" || ref.ExternalSkillID == "" {
			return favorites.Favorite{}, catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "externalProvider, externalSource, and externalSkillId are required.", nil)
		}
		refKey = "external:" + ref.ExternalProvider + ":" + ref.ExternalSource + ":" + ref.ExternalSkillID
	default:
		return favorites.Favorite{}, catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "refType must be catalog or external.", nil)
	}
	if snapshot == nil {
		snapshot = json.RawMessage(`{}`)
	}
	return favorites.Favorite{
		UserID:           current.UserID,
		ContextType:      contextType,
		OrganizationID:   organizationID,
		RefKey:           refKey,
		RefType:          refType,
		PackageName:      packageName,
		ExternalProvider: ref.ExternalProvider,
		ExternalSource:   ref.ExternalSource,
		ExternalSkillID:  ref.ExternalSkillID,
		Snapshot:         snapshot,
	}, nil
}

func (s *Server) markFavoriteAvailability(ctx context.Context, current auth.Principal, items []favorites.Favorite) []favorites.Favorite {
	for index := range items {
		item := &items[index]
		item.Available = true
		if item.RefType != "catalog" || item.PackageName == "" || s.accessStore == nil || s.access == nil {
			continue
		}
		record, err := s.accessStore.GetSkillAccess(ctx, item.PackageName)
		if errors.Is(err, access.ErrNotFound) {
			continue
		}
		if err != nil {
			item.Available = false
			continue
		}
		if !s.access.Decide(ctx, current, access.ActionRead, record).Allowed {
			item.Available = false
			item.PackageName = ""
			item.Snapshot = nil
		}
	}
	return items
}

func requireFavoriteUser(context *gin.Context) auth.Principal {
	current := auth.PrincipalFromContext(context.Request.Context())
	if !current.IsUser() {
		platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(http.StatusUnauthorized, "unauthorized", "Authentication is required.", nil))
		return auth.Principal{}
	}
	return current
}
