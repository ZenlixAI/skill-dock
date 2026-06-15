package router

import (
	"github.com/gin-gonic/gin"

	governancehttp "skill_registry/internal/governance/http"
	systemhttp "skill_registry/internal/system/http"
)

func registerSystemRoutes(engine *gin.Engine, config Config) {
	systemhttp.RegisterRoutes(engine, systemhttp.Config{
		Healthz: config.Healthz,
		Readyz:  config.Readyz,
		Metrics: config.Metrics,
	})
}

func registerGovernanceRoutes(engine *gin.Engine, config Config) {
	governancehttp.RegisterRoutes(engine, governancehttp.Config{
		ServesReadRoutes:  config.ServesReadRoutes,
		ServesWriteRoutes: config.ServesWriteRoutes,
		GovernanceChanges: config.GovernanceChanges,
		SkillGovernance:   config.SkillGovernance,
		ReleaseRecord:     config.ReleaseRecord,
		VersionGovernance: config.VersionGovernance,
	})
}
