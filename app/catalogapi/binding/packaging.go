package binding

import (
	"github.com/gin-gonic/gin"

	"skill_registry/business/catalog"
)

type BundleDownloadQuery struct {
	WorkspacePath string `form:"workspacePath"`
	PackageName   string `form:"packageName"`
	Version       string `form:"version"`
}

func BindBundleDownloadQuery(context *gin.Context) (BundleDownloadQuery, *catalog.Error) {
	return bindQuery[BundleDownloadQuery](context)
}
