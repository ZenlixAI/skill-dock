package http

import (
	stdhttp "net/http"

	"github.com/gin-gonic/gin"
)

type Config struct {
	ServesReadRoutes  bool
	ServesWriteRoutes bool
	AuditEvents       gin.HandlerFunc
	Skills            gin.HandlerFunc
	SkillsSearch      gin.HandlerFunc
	CuratedSkills     gin.HandlerFunc
	SkillDetail       gin.HandlerFunc
	VersionList       gin.HandlerFunc
	VersionDetail     gin.HandlerFunc
	InstallTarget     gin.HandlerFunc
}

func RegisterRoutes(engine *gin.Engine, config Config) {
	if config.ServesWriteRoutes {
		registerHeadCompatibleRoute(engine, "/api/v1/audit-events", config.AuditEvents)
	}

	if !config.ServesReadRoutes {
		return
	}

	registerHeadCompatibleRoute(engine, "/api/v1/skills", config.Skills)
	registerHeadCompatibleRoute(engine, "/api/v1/skills/search", config.SkillsSearch)
	registerHeadCompatibleRoute(engine, "/api/v1/skills/curated", config.CuratedSkills)
	registerHeadCompatibleRoute(engine, "/api/v1/skills/:package", config.SkillDetail)
	registerHeadCompatibleRoute(engine, "/api/v1/skills/:package/versions", config.VersionList)
	registerHeadCompatibleRoute(engine, "/api/v1/skills/:package/versions/:version", config.VersionDetail)
	registerHeadCompatibleRoute(engine, "/api/v1/skills/:package/install-target", config.InstallTarget)
}

func registerHeadCompatibleRoute(engine *gin.Engine, path string, handler gin.HandlerFunc) {
	engine.Match([]string{stdhttp.MethodGet, stdhttp.MethodHead}, path, handler)
}
