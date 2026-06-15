package router

import (
	"github.com/gin-gonic/gin"

	platformhttp "skill_registry/internal/platform/http"
)

type Config struct {
	ServesReadRoutes  bool
	ServesWriteRoutes bool
	Middleware        []gin.HandlerFunc

	Healthz       gin.HandlerFunc
	Readyz        gin.HandlerFunc
	Metrics       gin.HandlerFunc
	Session       gin.HandlerFunc
	SessionLogin  gin.HandlerFunc
	SessionLogout gin.HandlerFunc
	AuditEvents   gin.HandlerFunc

	Skills            gin.HandlerFunc
	SkillsSearch      gin.HandlerFunc
	CuratedSkills     gin.HandlerFunc
	GovernanceChanges gin.HandlerFunc
	SkillDetail       gin.HandlerFunc
	VersionList       gin.HandlerFunc
	VersionDetail     gin.HandlerFunc
	InstallTarget     gin.HandlerFunc
	SkillGovernance   gin.HandlerFunc
	ReleaseRecord     gin.HandlerFunc
	VersionGovernance gin.HandlerFunc

	PackagingPrepare  gin.HandlerFunc
	PackagingValidate gin.HandlerFunc
	PackagingPack     gin.HandlerFunc
	PackagingPublish  gin.HandlerFunc
	PackagingCleanup  gin.HandlerFunc
	PackagingBundle   gin.HandlerFunc
}

func New(config Config) *gin.Engine {
	engine := platformhttp.NewEngine(config.Middleware)

	registerSystemRoutes(engine, config)
	registerSessionRoutes(engine, config)
	registerCatalogRoutes(engine, config)
	registerPackagingRoutes(engine, config)

	return engine
}
