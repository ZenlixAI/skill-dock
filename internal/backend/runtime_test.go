package backend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"skill_registry/internal/backend/httpserver"
	cataloginfra "skill_registry/internal/catalog/infra"
	packagingapplication "skill_registry/internal/packaging/application"
)

func TestNewRuntimeBuildsCatalogReadServiceAndHTTPServer(t *testing.T) {
	root := t.TempDir()

	runtime, err := NewRuntime(context.Background(), Config{
		Addr:                ":8080",
		DatabaseURL:         "file:" + filepath.Join(root, "runtime.db"),
		SharedWorkspaceRoot: filepath.Join(root, "workspaces"),
		ArtifactSource:      cataloginfra.NewStaticArtifactSource(nil),
		Server: httpserver.ServerConfig{
			ReaderToken: "reader-secret",
		},
		HTTPServer: httpserver.HTTPServerConfig{
			WriteTimeout: 7 * time.Second,
		},
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	if runtime.CatalogRead == nil {
		t.Fatal("expected catalog read service to be wired")
	}
	if runtime.CatalogWrite == nil {
		t.Fatal("expected catalog write service to be wired")
	}
	if runtime.Server == nil {
		t.Fatal("expected catalog server to be wired")
	}
	if runtime.Handler == nil {
		t.Fatal("expected HTTP handler to be wired")
	}
	if runtime.HTTPServer == nil {
		t.Fatal("expected HTTP server to be wired")
	}
	if runtime.HTTPServer.Handler != runtime.Handler {
		t.Fatal("expected HTTP server to reuse runtime handler")
	}
	if runtime.HTTPServer.WriteTimeout != 7*time.Second {
		t.Fatalf("expected configured write timeout, got %v", runtime.HTTPServer.WriteTimeout)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	recorder := httptest.NewRecorder()

	runtime.Handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}
}

func TestNewRuntimeBuildsExplicitCatalogServicesInsteadOfReusingRepositoryProvider(t *testing.T) {
	root := t.TempDir()

	runtime, err := NewRuntime(context.Background(), Config{
		DatabaseURL:         "file:" + filepath.Join(root, "runtime.db"),
		SharedWorkspaceRoot: filepath.Join(root, "workspaces"),
		ArtifactSource:      cataloginfra.NewStaticArtifactSource(nil),
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	if runtime.Repository == nil {
		t.Fatal("expected runtime repository")
	}
}

func TestNewRuntimeSeedsSampleCatalogOnEmptySQLStore(t *testing.T) {
	root := t.TempDir()

	runtime, err := NewRuntime(context.Background(), Config{
		DatabaseURL:         "file:" + filepath.Join(root, "runtime.db"),
		SharedWorkspaceRoot: filepath.Join(root, "workspaces"),
		ArtifactSource:      cataloginfra.NewStaticArtifactSource(cataloginfra.SampleData().Artifacts),
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	skill, err := runtime.Repository.GetSkill(context.Background(), "@platform.skill/pdf-skill")
	if err != nil {
		t.Fatalf("expected seeded skill, got error: %v", err)
	}
	if skill.PackageName != "@platform.skill/pdf-skill" {
		t.Fatalf("expected seeded skill package, got %q", skill.PackageName)
	}
}

func TestNewPackagingServiceReturnsInternalApplicationService(t *testing.T) {
	service := packagingapplication.NewServiceWithWorkspaceStore(t.TempDir(), nil)
	if service == nil {
		t.Fatal("expected packaging service")
	}

	var _ *packagingapplication.Service = service
}
