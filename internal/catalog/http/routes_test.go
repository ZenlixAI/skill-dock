package http

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutesMountsSkillsEndpoint(t *testing.T) {
	engine := gin.New()
	RegisterRoutes(engine, Config{
		ServesReadRoutes: true,
		Skills: func(ctx *gin.Context) {
			ctx.Status(stdhttp.StatusNoContent)
		},
	})

	req := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/skills", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != stdhttp.StatusNoContent {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusNoContent, rec.Code)
	}
}
