package router

import "github.com/gin-gonic/gin"

func registerCatalogRoutes(engine *gin.Engine, config Config) {
	if config.ServesWriteRoutes {
		engine.GET("/api/v1/audit-events", config.AuditEvents)
	}

	if config.ServesReadRoutes {
		engine.GET("/api/v1/skills", config.Skills)
		engine.GET("/api/v1/skills/search", config.SkillsSearch)
		engine.GET("/api/v1/skills/curated", config.CuratedSkills)
		engine.GET("/api/v1/governance-changes", config.GovernanceChanges)
		engine.GET("/api/v1/skills/:package", config.SkillDetail)
		engine.GET("/api/v1/skills/:package/versions", config.VersionList)
		engine.GET("/api/v1/skills/:package/versions/:version", config.VersionDetail)
		engine.GET("/api/v1/skills/:package/install-target", config.InstallTarget)
	}

	if config.ServesWriteRoutes {
		engine.POST("/api/v1/skills/:package/governance-changes", config.SkillGovernance)
		engine.PUT("/api/v1/skills/:package/versions/:version/release-record", config.ReleaseRecord)
		engine.POST("/api/v1/skills/:package/versions/:version/governance-changes", config.VersionGovernance)
	}
}
