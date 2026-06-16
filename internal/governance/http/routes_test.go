package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	catalog "skill_registry/internal/catalog/domain"
	"skill_registry/internal/platform/auth"
)

func TestGovernanceChangesUsesGovernanceHandlerBehavior(t *testing.T) {
	t.Helper()

	var gotQuery catalog.GovernanceQuery
	var gotPrincipal auth.Principal

	handlers := New(Handlers{
		AuthorizePackage: func(http.ResponseWriter, *http.Request, string) bool {
			return true
		},
		ServeCachedReadJSON: func(writer http.ResponseWriter, request *http.Request, ttl time.Duration, load func() (any, *catalog.Error)) {
			payload, err := load()
			if err != nil {
				t.Fatalf("unexpected catalog error: %+v", err)
			}
			writer.WriteHeader(http.StatusOK)
			_ = payload
		},
		ListGovernance: func(_ context.Context, principal auth.Principal, query catalog.GovernanceQuery) (catalog.GovernancePage, *catalog.Error) {
			gotPrincipal = principal
			gotQuery = query
			return catalog.GovernancePage{}, nil
		},
	})

	engine := gin.New()
	engine.GET("/api/v1/governance-changes", func(ctx *gin.Context) {
		ctx.Request = ctx.Request.WithContext(auth.ContextWithPrincipal(ctx.Request.Context(), auth.Principal{
			Role:  "publisher",
			Actor: "reviewer",
		}))
		handlers.GovernanceChanges(ctx)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/governance-changes?package_name=%40platform.skill%2Fpdf-skill&target_type=version&version=1.2.0&pageSize=25&cursor=3", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if gotPrincipal.Actor != "reviewer" {
		t.Fatalf("expected principal actor reviewer, got %q", gotPrincipal.Actor)
	}
	if gotQuery.PackageName != "@platform.skill/pdf-skill" {
		t.Fatalf("expected package query to be decoded, got %q", gotQuery.PackageName)
	}
	if gotQuery.TargetType != string(catalog.GovernanceTargetVersion) {
		t.Fatalf("expected target type %q, got %q", catalog.GovernanceTargetVersion, gotQuery.TargetType)
	}
	if gotQuery.Version != "1.2.0" {
		t.Fatalf("expected version 1.2.0, got %q", gotQuery.Version)
	}
	if gotQuery.PageSize != 25 {
		t.Fatalf("expected page size 25, got %d", gotQuery.PageSize)
	}
	if gotQuery.Cursor != 3 {
		t.Fatalf("expected cursor 3, got %d", gotQuery.Cursor)
	}
}
