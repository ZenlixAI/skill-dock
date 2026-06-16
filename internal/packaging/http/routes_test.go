package http

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutesMountsPrepareEndpoint(t *testing.T) {
	engine := gin.New()
	RegisterRoutes(engine, Config{
		ServesWriteRoutes: true,
		ImportReview: func(ctx *gin.Context) {
			ctx.Status(stdhttp.StatusNoContent)
		},
		Publish: func(ctx *gin.Context) {
			ctx.Status(stdhttp.StatusNoContent)
		},
	})

	req := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/skill-packaging/import-review", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != stdhttp.StatusNoContent {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusNoContent, rec.Code)
	}
}

func TestRegisterRoutesDoesNotMountLegacyWorkspaceEndpoints(t *testing.T) {
	engine := gin.New()
	RegisterRoutes(engine, Config{
		ServesWriteRoutes: true,
		ImportReview: func(ctx *gin.Context) {
			ctx.Status(stdhttp.StatusNoContent)
		},
		Publish: func(ctx *gin.Context) {
			ctx.Status(stdhttp.StatusNoContent)
		},
	})

	for _, requestPath := range []string{
		"/api/v1/skill-packaging/prepare",
		"/api/v1/skill-packaging/validate",
		"/api/v1/skill-packaging/pack",
		"/api/v1/skill-packaging/cleanup",
		"/api/v1/skill-packaging/bundle",
	} {
		req := httptest.NewRequest(stdhttp.MethodPost, requestPath, nil)
		if requestPath == "/api/v1/skill-packaging/bundle" {
			req = httptest.NewRequest(stdhttp.MethodGet, requestPath, nil)
		}
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code != stdhttp.StatusNotFound {
			t.Fatalf("expected %s to be unmounted with 404, got %d", requestPath, rec.Code)
		}
	}
}
