package http

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
	var query BundleDownloadQuery
	if err := context.ShouldBindQuery(&query); err != nil {
		return query, catalog.NewError(400, "invalid_request", "query parameters are invalid.", map[string]any{"error": err.Error()})
	}
	return query, nil
}
