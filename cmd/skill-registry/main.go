package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"skill_registry/app/catalogapi"
	"skill_registry/internal/backend"
	catalog "skill_registry/internal/catalog/domain"
	cataloginfra "skill_registry/internal/catalog/infra"
	artifactadapter "skill_registry/internal/platform/artifact"
	"skill_registry/internal/platform/auth"
)

func main() {
	runtime, err := backend.NewRuntime(context.Background(), configuredRuntimeConfig())
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("skill registry listening on %s", runtime.HTTPServer.Addr)
	if err := serveHTTPServer(runtime.HTTPServer); err != nil {
		log.Fatal(err)
	}
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func configuredArtifactSource() catalog.ArtifactSource {
	if path := os.Getenv("SKILL_REGISTRY_ARTIFACTS_PATH"); path != "" {
		return artifactadapter.NewJSONArtifactSource(path)
	}
	if registryURL := os.Getenv("SKILL_REGISTRY_NPM_REGISTRY_URL"); registryURL != "" {
		return artifactadapter.NewNPMRegistryArtifactSource(registryURL, nil)
	}
	return cataloginfra.NewStaticArtifactSource(nil)
}

func configuredRepositoryOptions(artifactSource catalog.ArtifactSource) cataloginfra.SQLRepositoryOptions {
	return cataloginfra.SQLRepositoryOptions{
		ArtifactSource: artifactSource,
		DisableSeed:    true,
	}
}

func configuredRuntimeConfig() backend.Config {
	artifactSource := configuredArtifactSource()

	return backend.Config{
		Addr:                envOrDefault("SKILL_REGISTRY_ADDR", ":8080"),
		DatabaseURL:         envOrDefault("SKILL_REGISTRY_DATABASE_URL", "file:"+filepath.Join("data", "skill-registry.db")),
		SharedWorkspaceRoot: envOrDefault("SKILL_REGISTRY_SHARED_WORKSPACE_ROOT", filepath.Join("data", "packaging-workspaces")),
		ArtifactSource:      artifactSource,
		RepositoryOptions:   configuredRepositoryOptions(artifactSource),
		Server: catalogapi.ServerConfig{
			Mode:                            catalogapi.ServerMode(os.Getenv("SKILL_REGISTRY_SERVER_MODE")),
			ReadListCacheTTL:                envDurationOrDefault("SKILL_REGISTRY_READ_CACHE_LIST_TTL", 2*time.Second),
			ReadDetailCacheTTL:              envDurationOrDefault("SKILL_REGISTRY_READ_CACHE_DETAIL_TTL", 5*time.Second),
			ReadMaxInFlight:                 envIntOrDefault("SKILL_REGISTRY_READ_MAX_IN_FLIGHT", 64),
			WriteMaxInFlight:                envIntOrDefault("SKILL_REGISTRY_WRITE_MAX_IN_FLIGHT", 4),
			ReaderToken:                     os.Getenv("SKILL_REGISTRY_READER_TOKEN"),
			AuthoringToken:                  os.Getenv("SKILL_REGISTRY_AUTHORING_TOKEN"),
			PublisherToken:                  os.Getenv("SKILL_REGISTRY_PUBLISHER_TOKEN"),
			NPMRegistryURL:                  os.Getenv("SKILL_REGISTRY_NPM_REGISTRY_URL"),
			NPMRegistryPublishAuthorization: configuredNPMPublishAuthorization(),
			PublicNPMRegistryURL:            os.Getenv("SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL"),
			AuthTokens:                      configuredAuthTokens(),
			SessionSecret:                   os.Getenv("SKILL_REGISTRY_SESSION_SECRET"),
			SessionUsers:                    configuredSessionUsers(),
			Logger:                          log.New(os.Stderr, "", 0),
		},
		HTTPServer: catalogapi.HTTPServerConfig{
			ReadHeaderTimeout: envDurationOrDefault("SKILL_REGISTRY_HTTP_READ_HEADER_TIMEOUT", 2*time.Second),
			ReadTimeout:       envDurationOrDefault("SKILL_REGISTRY_HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:      envDurationOrDefault("SKILL_REGISTRY_HTTP_WRITE_TIMEOUT", 60*time.Second),
			IdleTimeout:       envDurationOrDefault("SKILL_REGISTRY_HTTP_IDLE_TIMEOUT", 60*time.Second),
		},
		DBPool: backend.DBPoolConfig{
			MaxOpenConns:    envIntOrDefault("SKILL_REGISTRY_DB_MAX_OPEN_CONNS", 20),
			MaxIdleConns:    envIntOrDefault("SKILL_REGISTRY_DB_MAX_IDLE_CONNS", 20),
			ConnMaxLifetime: envDurationOrDefault("SKILL_REGISTRY_DB_CONN_MAX_LIFETIME", time.Hour),
			ConnMaxIdleTime: envDurationOrDefault("SKILL_REGISTRY_DB_CONN_MAX_IDLE_TIME", 15*time.Minute),
		},
		IdempotencyTTL: 10 * time.Minute,
	}
}

func configuredAuthTokens() map[string]auth.TokenConfig {
	raw := os.Getenv("SKILL_REGISTRY_AUTH_TOKENS_JSON")
	if raw == "" {
		return nil
	}

	var authTokens map[string]auth.TokenConfig
	if err := json.Unmarshal([]byte(raw), &authTokens); err != nil {
		log.Fatalf("parse SKILL_REGISTRY_AUTH_TOKENS_JSON: %v", err)
	}
	return authTokens
}

func configuredSessionUsers() map[string]auth.SessionUserConfig {
	raw := os.Getenv("SKILL_REGISTRY_SESSION_USERS_JSON")
	if raw == "" {
		return nil
	}

	var sessionUsers map[string]auth.SessionUserConfig
	if err := json.Unmarshal([]byte(raw), &sessionUsers); err != nil {
		log.Fatalf("parse SKILL_REGISTRY_SESSION_USERS_JSON: %v", err)
	}
	return sessionUsers
}

func envIntOrDefault(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("parse %s: expected integer value, got %q", key, value)
	}
	return parsed
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		log.Fatalf("parse %s: expected duration value, got %q", key, value)
	}
	return parsed
}

func configuredNPMPublishAuthorization() string {
	if value := strings.TrimSpace(os.Getenv("SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION")); value != "" {
		return value
	}
	if token := strings.TrimSpace(os.Getenv("SKILL_REGISTRY_NPM_PUBLISH_TOKEN")); token != "" {
		return "Bearer " + token
	}
	return ""
}

func serveHTTPServer(httpServer *http.Server) error {
	if httpServer == nil {
		return nil
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httpServer.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		if err == nil || err == http.ErrServerClosed {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), envDurationOrDefault("SKILL_REGISTRY_HTTP_SHUTDOWN_TIMEOUT", 20*time.Second))
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return err
	}

	if err := <-serverErr; err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
