package response

import (
	"github.com/gin-gonic/gin"

	"skill_registry/app/catalogapi/common"
)

func WriteCatalogError(context *gin.Context, err error) {
	common.WriteCatalogError(context.Writer, context.Request, err)
}
