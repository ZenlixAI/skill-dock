package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerCatalogRoutes(engine *gin.Engine, config Config) {
	if config.ServesWriteRoutes {
		registerHeadCompatibleRoute(engine, "/api/v1/audit-events", config.AuditEvents)
	}

	if config.ServesReadRoutes {
		registerHeadCompatibleRoute(engine, "/api/v1/skills", config.Skills)
		registerHeadCompatibleRoute(engine, "/api/v1/skills/search", config.SkillsSearch)
		registerHeadCompatibleRoute(engine, "/api/v1/skills/curated", config.CuratedSkills)
		registerHeadCompatibleRoute(engine, "/api/v1/governance-changes", config.GovernanceChanges)
		registerHeadCompatibleRoute(engine, "/api/v1/skills/:package", config.SkillDetail)
		registerHeadCompatibleRoute(engine, "/api/v1/skills/:package/versions", config.VersionList)
		registerHeadCompatibleRoute(engine, "/api/v1/skills/:package/versions/:version", config.VersionDetail)
		registerHeadCompatibleRoute(engine, "/api/v1/skills/:package/install-target", config.InstallTarget)
	}

	if config.ServesWriteRoutes {
		engine.POST("/api/v1/skills/:package/governance-changes", config.SkillGovernance)
		engine.PUT("/api/v1/skills/:package/versions/:version/release-record", config.ReleaseRecord)
		engine.POST("/api/v1/skills/:package/versions/:version/governance-changes", config.VersionGovernance)
	}
}

func registerHeadCompatibleRoute(engine *gin.Engine, path string, handler gin.HandlerFunc) {
	engine.Match([]string{http.MethodGet, http.MethodHead}, path, handler)
}
