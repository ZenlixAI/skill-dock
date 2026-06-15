package response

import "github.com/gin-gonic/gin"

func WriteJSON(context *gin.Context, status int, payload any) {
	context.JSON(status, payload)
}
