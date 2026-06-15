package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"skill_registry/app/catalogapi"
	"skill_registry/business/catalog"
	artifactadapter "skill_registry/foundation/artifact"
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
	server := catalogapi.NewServer(catalogapi.ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
	})
	httpServer := catalogapi.NewHTTPServer(":8080", server.Handler(), catalogapi.HTTPServerConfig{
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

	if _, ok := source.(*artifactadapter.JSONArtifactSource); !ok {
		t.Fatalf("expected JSON artifact source, got %T", source)
	}
}

func TestConfiguredArtifactSourceUsesConfiguredNPMRegistry(t *testing.T) {
	t.Setenv("SKILL_REGISTRY_ARTIFACTS_PATH", "")
	t.Setenv("SKILL_REGISTRY_NPM_REGISTRY_URL", "https://registry.example.com")

	source := configuredArtifactSource()

	if _, ok := source.(*artifactadapter.NPMRegistryArtifactSource); !ok {
		t.Fatalf("expected NPM registry artifact source, got %T", source)
	}
}

func TestConfiguredArtifactSourceFallsBackToEmptyStaticSource(t *testing.T) {
	t.Setenv("SKILL_REGISTRY_ARTIFACTS_PATH", "")
	t.Setenv("SKILL_REGISTRY_NPM_REGISTRY_URL", "")

	source := configuredArtifactSource()

	staticSource, ok := source.(*catalog.StaticArtifactSource)
	if !ok {
		t.Fatalf("expected static artifact source, got %T", source)
	}

	_, err := staticSource.FindReleaseArtifact(context.Background(), "@platform.skill/pdf-skill", "1.2.0")
	if err == nil {
		t.Fatal("expected empty fallback artifact source to miss sample artifact")
	}

	var catalogErr *catalog.Error
	if !errors.As(err, &catalogErr) {
		t.Fatalf("expected catalog error, got %T", err)
	}
	if catalogErr.Status != 404 {
		t.Fatalf("expected 404 from empty static artifact source, got %d", catalogErr.Status)
	}
}

func TestProductionSeedDataIsAlwaysEmpty(t *testing.T) {
	data := productionSeedData()

	if len(data.Skills) != 0 || len(data.Governance) != 0 || len(data.Artifacts) != 0 {
		t.Fatalf("expected empty production seed data, got %+v", data)
	}
}

func TestProductionMockRegistryIsAlwaysDisabled(t *testing.T) {
	registry, disabled := productionMockRegistry()

	if registry != nil {
		t.Fatalf("expected no mock registry, got %T", registry)
	}
	if !disabled {
		t.Fatal("expected mock registry to stay disabled in production startup")
	}
}
