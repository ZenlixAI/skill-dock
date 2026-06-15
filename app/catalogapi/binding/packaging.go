package binding

import (
	"github.com/gin-gonic/gin"

	"skill_registry/business/catalog"
	internalpackaginghttp "skill_registry/internal/packaging/http"
)

type BundleDownloadQuery = internalpackaginghttp.BundleDownloadQuery

func BindBundleDownloadQuery(context *gin.Context) (BundleDownloadQuery, *catalog.Error) {
	return internalpackaginghttp.BindBundleDownloadQuery(context)
}
