package backend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"skill_registry/app/catalogapi"
	"skill_registry/business/catalog"
	packagingapplication "skill_registry/internal/packaging/application"
)

func TestNewRuntimeBuildsCatalogReadServiceAndHTTPServer(t *testing.T) {
	root := t.TempDir()

	runtime, err := NewRuntime(context.Background(), Config{
		Addr:                ":8080",
		DatabaseURL:         "file:" + filepath.Join(root, "runtime.db"),
		SharedWorkspaceRoot: filepath.Join(root, "workspaces"),
		ArtifactSource:      catalog.NewStaticArtifactSource(nil),
		Server: catalogapi.ServerConfig{
			ReaderToken: "reader-secret",
		},
		HTTPServer: catalogapi.HTTPServerConfig{
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
		ArtifactSource:      catalog.NewStaticArtifactSource(nil),
	})
	if err != nil {
		t.Fatalf("new runtime: %v", err)
	}

	provider, ok := any(runtime.Repository).(catalog.ApplicationServiceProvider)
	if !ok {
		t.Fatalf("expected runtime repository to expose compatibility provider, got %T", runtime.Repository)
	}

	if runtime.CatalogRead == provider.CatalogReadService() {
		t.Fatal("expected runtime to construct explicit read service instead of reusing repository provider")
	}
	if runtime.CatalogWrite == provider.CatalogWriteService() {
		t.Fatal("expected runtime to construct explicit write service instead of reusing repository provider")
	}
}

func TestNewPackagingServiceReturnsInternalApplicationService(t *testing.T) {
	service := newPackagingService(t.TempDir(), nil)
	if service == nil {
		t.Fatal("expected packaging service")
	}

	var _ *packagingapplication.Service = service
}
