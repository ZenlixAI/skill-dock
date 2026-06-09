package main

import (
	"os"
	"testing"
	"time"

	"skill_registry/app/catalogapi"
	"skill_registry/business/catalog"
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
