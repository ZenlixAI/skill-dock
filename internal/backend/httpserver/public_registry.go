package httpserver

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	catalog "skill_registry/internal/catalog/domain"
	internalcataloghttp "skill_registry/internal/catalog/http"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
)

const (
	publicRegistryBasePath       = "/api/v1/public/registry"
	publicExternalListCacheTTL   = 30 * time.Second
	publicExternalDetailCacheTTL = 10 * time.Minute
)

type publicRegistryCapabilities struct {
	ExternalSearch publicRegistryExternalCapability `json:"externalSearch"`
}

type publicRegistryExternalCapability struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
}

func (s *Server) registerPublicRegistryRoutes(engine *gin.Engine) {
	if engine == nil || !s.mode.servesReadRoutes() {
		return
	}

	engine.GET(publicRegistryBasePath+"/capabilities", withPublicRegistryCORS(s.publicRegistryCapabilities))
	engine.GET(publicRegistryBasePath+"/skills", withPublicRegistryCORS(s.publicRegistrySkills))
	engine.GET(publicRegistryBasePath+"/external/skills", withPublicRegistryCORS(s.publicRegistryExternalSkills))
	engine.GET(publicRegistryBasePath+"/external/skill", withPublicRegistryCORS(s.publicRegistryExternalSkill))
	engine.OPTIONS(publicRegistryBasePath+"/*path", withPublicRegistryCORS(func(context *gin.Context) {
		context.Status(http.StatusNoContent)
	}))

	// The shipped static site talks to the same route names through either the
	// Cloudflare Worker or this server (for self-hosted/Docker deployments).
	engine.GET("/api/skills-sh/capabilities", withPublicRegistryCORS(s.publicRegistryCapabilities))
	engine.GET("/api/skills-sh/search", withPublicRegistryCORS(s.publicRegistryExternalSearch))
	engine.GET("/api/skills-sh/browse", withPublicRegistryCORS(s.publicRegistryExternalBrowse))
	engine.GET("/api/skills-sh/skill", withPublicRegistryCORS(s.publicRegistryExternalSkill))
	engine.OPTIONS("/api/skills-sh/*path", withPublicRegistryCORS(func(context *gin.Context) {
		context.Status(http.StatusNoContent)
	}))
}

func withPublicRegistryCORS(handler gin.HandlerFunc) gin.HandlerFunc {
	return func(context *gin.Context) {
		context.Header("Access-Control-Allow-Origin", "*")
		context.Header("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		context.Header("Access-Control-Allow-Headers", "Content-Type")
		context.Header("Vary", "Origin")
		if context.Request.Method == http.MethodOptions {
			context.Status(http.StatusNoContent)
			return
		}
		handler(context)
	}
}

func (s *Server) publicRegistryCapabilities(context *gin.Context) {
	context.Header("Cache-Control", "no-store")
	platformhttp.WriteJSON(context.Writer, http.StatusOK, publicRegistryCapabilities{
		ExternalSearch: publicRegistryExternalCapability{
			Enabled:  s.externalSkillSearcher != nil,
			Provider: "skills.sh",
		},
	})
}

func (s *Server) publicRegistrySkills(context *gin.Context) {
	context.Header("Cache-Control", "public, max-age=30")
	query, err := internalcataloghttp.BindSkillsQuery(context)
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return
	}

	// Public registry reads intentionally ignore caller credentials and external
	// source switches. The public surface only exposes the catalog.
	query.Source = "catalog"
	query.IncludeExternal = ""
	query.IncludeRetired = ""

	s.serveCachedReadJSON(context.Writer, context.Request, s.readListCacheTTL, func() (any, *catalog.Error) {
		if s.listSkillsFunc == nil {
			return catalog.SkillsPage{}, nil
		}
		return s.listSkillsFunc(context.Request.Context(), auth.Principal{}, query)
	})
}

func (s *Server) publicRegistryExternalSkills(context *gin.Context) {
	context.Header("Cache-Control", "public, max-age=30")
	if s.externalSkillSearcher == nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, externalSearchDisabledError())
		return
	}

	limit := publicRegistryPositiveInt(context.Query("limit"), 12, 100)
	query := strings.TrimSpace(context.Query("q"))
	if query != "" && len([]rune(query)) < 2 {
		platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(
			http.StatusBadRequest,
			"invalid_request",
			"q must contain at least 2 characters.",
			map[string]any{"q": query},
		))
		return
	}
	pageNumber := publicRegistryPositiveInt(context.Query("page"), 1, 1000)
	view := context.DefaultQuery("view", "all-time")

	s.serveCachedReadJSON(context.Writer, context.Request, publicExternalListCacheTTL, func() (any, *catalog.Error) {
		if query != "" {
			page, err := s.externalSkillSearcher.SearchSkills(context.Request.Context(), catalog.ExternalSearchQuery{
				Query:    query,
				PageSize: limit,
			})
			if err != nil {
				return nil, publicExternalError(err)
			}
			return page, nil
		}

		page, err := s.externalSkillSearcher.ListSkills(context.Request.Context(), catalog.ExternalBrowseQuery{
			View:     view,
			Page:     pageNumber,
			PageSize: limit,
		})
		if err != nil {
			return nil, publicExternalError(err)
		}
		return page, nil
	})
}

func (s *Server) publicRegistryExternalSearch(context *gin.Context) {
	context.Header("Cache-Control", "public, max-age=30")
	if s.externalSkillSearcher == nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, externalSearchDisabledError())
		return
	}

	query := strings.TrimSpace(context.Query("q"))
	if len([]rune(query)) < 2 {
		platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(
			http.StatusBadRequest,
			"invalid_request",
			"q must contain at least 2 characters.",
			map[string]any{"q": query},
		))
		return
	}
	limit := publicRegistryPositiveInt(context.Query("limit"), 20, 100)

	s.serveCachedReadJSON(context.Writer, context.Request, publicExternalListCacheTTL, func() (any, *catalog.Error) {
		page, err := s.externalSkillSearcher.SearchSkills(context.Request.Context(), catalog.ExternalSearchQuery{
			Query:    query,
			PageSize: limit,
		})
		if err != nil {
			return nil, publicExternalError(err)
		}
		return page, nil
	})
}

func (s *Server) publicRegistryExternalBrowse(context *gin.Context) {
	context.Header("Cache-Control", "public, max-age=30")
	if s.externalSkillSearcher == nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, externalSearchDisabledError())
		return
	}

	limit := publicRegistryPositiveInt(context.Query("limit"), 12, 100)
	pageNumber := publicRegistryPositiveInt(context.Query("page"), 1, 1000)
	view := context.DefaultQuery("view", "all-time")

	s.serveCachedReadJSON(context.Writer, context.Request, publicExternalListCacheTTL, func() (any, *catalog.Error) {
		page, err := s.externalSkillSearcher.ListSkills(context.Request.Context(), catalog.ExternalBrowseQuery{
			View:     view,
			Page:     pageNumber,
			PageSize: limit,
		})
		if err != nil {
			return nil, publicExternalError(err)
		}
		return page, nil
	})
}

func (s *Server) publicRegistryExternalSkill(context *gin.Context) {
	context.Header("Cache-Control", "public, max-age=300")
	if s.externalSkillSearcher == nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, externalSearchDisabledError())
		return
	}

	source := strings.TrimSpace(context.Query("source"))
	skillID := strings.TrimSpace(context.Query("skillId"))
	if source == "" || skillID == "" || strings.Count(source, "/") != 1 || strings.Contains(source, "..") || strings.Contains(skillID, "/") {
		platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(
			http.StatusBadRequest,
			"invalid_request",
			"source and skillId must identify a skill.",
			map[string]any{"source": source, "skillId": skillID},
		))
		return
	}

	s.serveCachedReadJSON(context.Writer, context.Request, publicExternalDetailCacheTTL, func() (any, *catalog.Error) {
		detail, err := s.externalSkillSearcher.GetSkill(context.Request.Context(), source, skillID)
		if err != nil {
			return nil, publicExternalError(err)
		}
		return detail, nil
	})
}

func externalSearchDisabledError() *catalog.Error {
	return catalog.NewError(
		http.StatusNotFound,
		"external_search_disabled",
		"External skill search is not enabled for this registry.",
		nil,
	)
}

func publicExternalError(_ error) *catalog.Error {
	return catalog.NewError(
		http.StatusBadGateway,
		"external_search_unavailable",
		"The external skill directory is temporarily unavailable.",
		nil,
	)
}

func publicRegistryPositiveInt(raw string, fallback int, maximum int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	if maximum > 0 && value > maximum {
		return maximum
	}
	return value
}
