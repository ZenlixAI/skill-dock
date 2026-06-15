package router

import (
	"github.com/gin-gonic/gin"

	cataloghttp "skill_registry/internal/catalog/http"
)

func registerCatalogRoutes(engine *gin.Engine, config Config) {
	cataloghttp.RegisterRoutes(engine, cataloghttp.Config{
		ServesReadRoutes:  config.ServesReadRoutes,
		ServesWriteRoutes: config.ServesWriteRoutes,
		AuditEvents:       config.AuditEvents,
		Skills:            config.Skills,
		SkillsSearch:      config.SkillsSearch,
		CuratedSkills:     config.CuratedSkills,
		SkillDetail:       config.SkillDetail,
		VersionList:       config.VersionList,
		VersionDetail:     config.VersionDetail,
		InstallTarget:     config.InstallTarget,
	})
}
