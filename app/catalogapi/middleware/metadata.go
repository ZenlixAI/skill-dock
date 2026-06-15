package middleware

import (
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"

	"skill_registry/app/catalogapi/common"
)

func WithRequestMetadata(requestID *atomic.Uint64) gin.HandlerFunc {
	return func(context *gin.Context) {
		nextID := uint64(1)
		if requestID != nil {
			nextID = requestID.Add(1)
		}
		currentRequestID := "req_" + strconv.FormatUint(nextID, 10)
		context.Header("X-Request-Id", currentRequestID)

		traceID := strings.TrimSpace(context.GetHeader("X-Trace-Id"))
		if traceID == "" {
			traceID = "trace_" + strings.TrimPrefix(currentRequestID, "req_")
		}
		context.Header("X-Trace-Id", traceID)

		context.Request = context.Request.WithContext(common.ContextWithRequestMetadata(context.Request.Context(), currentRequestID, traceID))
		context.Next()
	}
}
