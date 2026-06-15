package http

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewEnginePreservesRawPathSettingsAndAppliesMiddleware(t *testing.T) {
	engine := NewEngine([]gin.HandlerFunc{
		func(context *gin.Context) {
			context.Header("X-Engine-Middleware", "applied")
			context.Next()
		},
	})

	if !engine.UseRawPath {
		t.Fatal("expected UseRawPath enabled")
	}
	if engine.UnescapePathValues {
		t.Fatal("expected UnescapePathValues disabled")
	}
	if !engine.HandleMethodNotAllowed {
		t.Fatal("expected HandleMethodNotAllowed enabled")
	}
	if engine.RedirectTrailingSlash {
		t.Fatal("expected RedirectTrailingSlash disabled")
	}
	if engine.RedirectFixedPath {
		t.Fatal("expected RedirectFixedPath disabled")
	}

	engine.GET("/api/v1/skills/:packageName", func(context *gin.Context) {
		context.Status(stdhttp.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(stdhttp.MethodGet, "/api/v1/skills/%40platform.skill%2Fpdf-skill", nil)
	engine.ServeHTTP(recorder, request)

	if got := recorder.Header().Get("X-Engine-Middleware"); got != "applied" {
		t.Fatalf("expected middleware header, got %q", got)
	}
	if recorder.Code != stdhttp.StatusNoContent {
		t.Fatalf("expected status %d, got %d", stdhttp.StatusNoContent, recorder.Code)
	}
}
