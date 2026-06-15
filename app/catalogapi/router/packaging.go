package router

import (
	"github.com/gin-gonic/gin"

	packaginghttp "skill_registry/internal/packaging/http"
)

func registerPackagingRoutes(engine *gin.Engine, config Config) {
	packaginghttp.RegisterRoutes(engine, packaginghttp.Config{
		ServesWriteRoutes: config.ServesWriteRoutes,
		Prepare:           config.PackagingPrepare,
		Validate:          config.PackagingValidate,
		Pack:              config.PackagingPack,
		Publish:           config.PackagingPublish,
		Cleanup:           config.PackagingCleanup,
		Bundle:            config.PackagingBundle,
	})
}
