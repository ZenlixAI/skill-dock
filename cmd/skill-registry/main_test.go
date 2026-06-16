package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"skill_registry/internal/backend"
	"skill_registry/internal/backend/httpserver"
	catalogdomain "skill_registry/internal/catalog/domain"
	cataloginfra "skill_registry/internal/catalog/infra"
	platformartifact "skill_registry/internal/platform/artifact"
	platformauth "skill_registry/internal/platform/auth"
)

func TestEnvIntOrDefaultUsesFallbackForEmptyValue(t *testing.T) {
	t.Setenv("SKILL_REGISTRY_DB_MAX_OPEN_CONNS", "")

	if got := envIntOrDefault("SKILL_REGISTRY_DB_MAX_OPEN_CONNS", 17); got != 17 {
		t.Fatalf("expected fallback 17, got %d", got)
	}
}

func TestEnvDurationOrDefaultParsesConfiguredDuration(t *testing.T) {
	t.Setenv("SKILL_REGISTRY_HTTP_IDLE_TIMEOUT", "45s")

	if got := envDurationOrDefault("SKILL_REGISTRY_HTTP_IDLE_TIMEOUT", time.Second); got != 45*time.Second {
		t.Fatalf("expected 45s, got %v", got)
	}
}

func TestConfiguredHTTPServerBuildsTimeouts(t *testing.T) {
	server := httpserver.NewServer(httpserver.ServerConfig{
		Repository: cataloginfra.NewMemoryRepository(cataloginfra.SampleData()),
	})
	httpServer := httpserver.NewHTTPServer(":8080", server.Handler(), httpserver.HTTPServerConfig{
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	})

	if httpServer.ReadHeaderTimeout != 2*time.Second {
		t.Fatalf("expected read header timeout, got %v", httpServer.ReadHeaderTimeout)
	}
	if httpServer.ReadTimeout != 5*time.Second {
		t.Fatalf("expected read timeout, got %v", httpServer.ReadTimeout)
	}
	if httpServer.WriteTimeout != 10*time.Second {
		t.Fatalf("expected write timeout, got %v", httpServer.WriteTimeout)
	}
	if httpServer.IdleTimeout != 30*time.Second {
		t.Fatalf("expected idle timeout, got %v", httpServer.IdleTimeout)
	}
}

func TestEnvDurationOrDefaultUsesFallbackForEmptyValue(t *testing.T) {
	const key = "SKILL_REGISTRY_HTTP_WRITE_TIMEOUT"
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset env: %v", err)
	}

	if got := envDurationOrDefault(key, 12*time.Second); got != 12*time.Second {
		t.Fatalf("expected fallback 12s, got %v", got)
	}
}

func TestConfiguredArtifactSourceUsesConfiguredArtifactsPath(t *testing.T) {
	t.Setenv("SKILL_REGISTRY_ARTIFACTS_PATH", "testdata/artifacts.json")
	t.Setenv("SKILL_REGISTRY_NPM_REGISTRY_URL", "https://registry.example.com")

	source := configuredArtifactSource()

	if _, ok := source.(*platformartifact.JSONArtifactSource); !ok {
		t.Fatalf("expected JSON artifact source, got %T", source)
	}
}

func TestConfiguredAuthTokensUsePlatformTypes(t *testing.T) {
	t.Setenv("SKILL_REGISTRY_AUTH_TOKENS_JSON", `{"reader-token":{"role":"reader","actor":"alice"}}`)

	tokens := configuredAuthTokens()
	if _, ok := tokens["reader-token"]; !ok {
		t.Fatal("expected configured token")
	}

	var _ platformauth.TokenConfig = tokens["reader-token"]
}

func TestConfiguredArtifactSourceUsesConfiguredNPMRegistry(t *testing.T) {
	t.Setenv("SKILL_REGISTRY_ARTIFACTS_PATH", "")
	t.Setenv("SKILL_REGISTRY_NPM_REGISTRY_URL", "https://registry.example.com")

	source := configuredArtifactSource()

	if _, ok := source.(*platformartifact.NPMRegistryArtifactSource); !ok {
		t.Fatalf("expected NPM registry artifact source, got %T", source)
	}
}

func TestConfiguredArtifactSourceReturnsPlatformAdapter(t *testing.T) {
	t.Setenv("SKILL_REGISTRY_NPM_REGISTRY_URL", "https://registry.example.com")

	if _, ok := configuredArtifactSource().(*platformartifact.NPMRegistryArtifactSource); !ok {
		t.Fatalf("expected internal platform artifact adapter, got %T", configuredArtifactSource())
	}
}

func TestConfiguredArtifactSourceFallsBackToEmptyStaticSource(t *testing.T) {
	t.Setenv("SKILL_REGISTRY_ARTIFACTS_PATH", "")
	t.Setenv("SKILL_REGISTRY_NPM_REGISTRY_URL", "")

	source := configuredArtifactSource()

	staticSource, ok := source.(*cataloginfra.StaticArtifactSource)
	if !ok {
		t.Fatalf("expected static artifact source, got %T", source)
	}

	_, err := staticSource.FindReleaseArtifact(context.Background(), "@platform.skill/pdf-skill", "1.2.0")
	if err == nil {
		t.Fatal("expected empty fallback artifact source to miss sample artifact")
	}

	var catalogErr *catalogdomain.Error
	if !errors.As(err, &catalogErr) {
		t.Fatalf("expected catalog error, got %T", err)
	}
	if catalogErr.Status != 404 {
		t.Fatalf("expected 404 from empty static artifact source, got %d", catalogErr.Status)
	}
}

func TestConfiguredRepositoryOptionsDisableSeedAndPreserveArtifactSource(t *testing.T) {
	artifactSource := cataloginfra.NewStaticArtifactSource(nil)

	options := configuredRepositoryOptions(artifactSource)

	if options.ArtifactSource != artifactSource {
		t.Fatalf("expected repository options to preserve artifact source, got %T", options.ArtifactSource)
	}
	if !options.DisableSeed {
		t.Fatal("expected production repository options to disable seeding")
	}
}

func TestConfiguredRuntimeConfigUsesProductionOnlyStartupWiring(t *testing.T) {
	t.Setenv("SKILL_REGISTRY_DATABASE_URL", "file:test.db")
	t.Setenv("SKILL_REGISTRY_SHARED_WORKSPACE_ROOT", "packaging-root")

	config := configuredRuntimeConfig()

	if config.DatabaseURL != "file:test.db" {
		t.Fatalf("expected database URL to be preserved, got %q", config.DatabaseURL)
	}
	if config.SharedWorkspaceRoot != "packaging-root" {
		t.Fatalf("expected shared workspace root to be preserved, got %q", config.SharedWorkspaceRoot)
	}
	if config.ArtifactSource == nil {
		t.Fatal("expected runtime config to carry an artifact source")
	}
	if config.Server.Repository != nil {
		t.Fatalf("expected runtime config to defer repository assembly, got %T", config.Server.Repository)
	}
	if config.Server.Packaging != nil {
		t.Fatalf("expected runtime config to defer packaging assembly, got %T", config.Server.Packaging)
	}
	if config.Server.Idempotency != nil {
		t.Fatalf("expected runtime config to defer idempotency assembly, got %T", config.Server.Idempotency)
	}
	if config.Server.Audit != nil {
		t.Fatalf("expected runtime config to defer audit assembly, got %T", config.Server.Audit)
	}
	if config.Server.Logger == nil {
		t.Fatal("expected runtime config to configure a logger")
	}
	if config.IdempotencyTTL != 10*time.Minute {
		t.Fatalf("expected default idempotency TTL, got %v", config.IdempotencyTTL)
	}

	var _ backend.Config = config
}
