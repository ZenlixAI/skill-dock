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

	"skill_registry/internal/backend"
	"skill_registry/internal/backend/httpserver"
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
	artifactReadURL := strings.TrimSpace(os.Getenv("SKILL_REGISTRY_ARTIFACT_READ_URL"))
	if artifactReadURL == "" {
		artifactReadURL = strings.TrimSpace(os.Getenv("SKILL_REGISTRY_NPM_REGISTRY_URL"))
	}
	if artifactReadURL != "" {
		artifactReadAuthorization := strings.TrimSpace(os.Getenv("SKILL_REGISTRY_ARTIFACT_READ_AUTHORIZATION"))
		if artifactReadAuthorization == "" {
			artifactReadAuthorization = strings.TrimSpace(os.Getenv("SKILL_REGISTRY_NPM_READ_AUTHORIZATION"))
			if artifactReadAuthorization == "" {
				artifactReadAuthorization = strings.TrimSpace(os.Getenv("SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION"))
				if artifactReadAuthorization == "" {
					if token := strings.TrimSpace(os.Getenv("SKILL_REGISTRY_NPM_PUBLISH_TOKEN")); token != "" {
						artifactReadAuthorization = "Bearer " + token
					}
				}
			}
		}
		return artifactadapter.NewAuthenticatedNPMRegistryArtifactSource(artifactReadURL, artifactReadAuthorization, nil)
	}
	return cataloginfra.NewStaticArtifactSource(nil)
}

func configuredRepositoryOptions(artifactSource catalog.ArtifactSource) cataloginfra.SQLRepositoryOptions {
	return cataloginfra.SQLRepositoryOptions{
		ArtifactSource: artifactSource,
		DisableSeed:    envBoolOrDefault("SKILL_REGISTRY_DISABLE_SEED", true),
	}
}

func configuredExternalSkillSearcher() catalog.ExternalSkillSearcher {
	skillsShURL := strings.TrimSpace(os.Getenv("SKILL_REGISTRY_SKILLS_SH_URL"))
	if skillsShURL == "" {
		skillsShURL = "https://skills.sh"
	}
	return cataloginfra.NewSkillsShSearchSource(skillsShURL, nil)
}

func configuredRuntimeConfig() backend.Config {
	artifactReadURL := strings.TrimSpace(os.Getenv("SKILL_REGISTRY_ARTIFACT_READ_URL"))
	if artifactReadURL == "" {
		artifactReadURL = strings.TrimSpace(os.Getenv("SKILL_REGISTRY_NPM_REGISTRY_URL"))
	}
	artifactReadAuthorization := strings.TrimSpace(os.Getenv("SKILL_REGISTRY_ARTIFACT_READ_AUTHORIZATION"))
	if artifactReadAuthorization == "" {
		artifactReadAuthorization = strings.TrimSpace(os.Getenv("SKILL_REGISTRY_NPM_READ_AUTHORIZATION"))
		if artifactReadAuthorization == "" {
			artifactReadAuthorization = strings.TrimSpace(os.Getenv("SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION"))
			if artifactReadAuthorization == "" {
				if token := strings.TrimSpace(os.Getenv("SKILL_REGISTRY_NPM_PUBLISH_TOKEN")); token != "" {
					artifactReadAuthorization = "Bearer " + token
				}
			}
		}
	}
	artifactPublishAuthorization := strings.TrimSpace(os.Getenv("SKILL_REGISTRY_ARTIFACT_PUBLISH_AUTHORIZATION"))
	if artifactPublishAuthorization == "" {
		artifactPublishAuthorization = strings.TrimSpace(os.Getenv("SKILL_REGISTRY_NPM_PUBLISH_AUTHORIZATION"))
		if artifactPublishAuthorization == "" {
			if token := strings.TrimSpace(os.Getenv("SKILL_REGISTRY_NPM_PUBLISH_TOKEN")); token != "" {
				artifactPublishAuthorization = "Bearer " + token
			}
		}
	}
	publicInstallURL := strings.TrimSpace(os.Getenv("SKILL_REGISTRY_PUBLIC_INSTALL_URL"))
	if publicInstallURL == "" {
		publicInstallURL = strings.TrimSpace(os.Getenv("SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL"))
	}
	artifactSource := configuredArtifactSource()

	installTargetMode := httpserver.InstallTargetMode(os.Getenv("SKILL_REGISTRY_INSTALL_TARGET_MODE"))
	if strings.TrimSpace(string(installTargetMode)) == "" {
		installTargetMode = httpserver.InstallTargetModeProxy
	}

	return backend.Config{
		Addr:                envOrDefault("SKILL_REGISTRY_ADDR", ":8080"),
		DatabaseURL:         envOrDefault("SKILL_REGISTRY_DATABASE_URL", "file:"+filepath.Join("data", "skill-registry.db")),
		SharedWorkspaceRoot: envOrDefault("SKILL_REGISTRY_SHARED_WORKSPACE_ROOT", filepath.Join("data", "packaging-workspaces")),
		ArtifactSource:      artifactSource,
		RepositoryOptions:   configuredRepositoryOptions(artifactSource),
		Server: httpserver.ServerConfig{
			Mode:                            httpserver.ServerMode(os.Getenv("SKILL_REGISTRY_SERVER_MODE")),
			InstallTargetMode:               installTargetMode,
			ReadListCacheTTL:                envDurationOrDefault("SKILL_REGISTRY_READ_CACHE_LIST_TTL", 2*time.Second),
			ReadDetailCacheTTL:              envDurationOrDefault("SKILL_REGISTRY_READ_CACHE_DETAIL_TTL", 5*time.Second),
			ReadMaxInFlight:                 envIntOrDefault("SKILL_REGISTRY_READ_MAX_IN_FLIGHT", 64),
			WriteMaxInFlight:                envIntOrDefault("SKILL_REGISTRY_WRITE_MAX_IN_FLIGHT", 4),
			ReaderToken:                     os.Getenv("SKILL_REGISTRY_READER_TOKEN"),
			AuthoringToken:                  os.Getenv("SKILL_REGISTRY_AUTHORING_TOKEN"),
			PublisherToken:                  os.Getenv("SKILL_REGISTRY_PUBLISHER_TOKEN"),
			NPMRegistryURL:                  artifactReadURL,
			NPMRegistryReadAuthorization:    artifactReadAuthorization,
			NPMRegistryPublishAuthorization: artifactPublishAuthorization,
			PublicNPMRegistryURL:            publicInstallURL,
			AuthTokens:                      configuredAuthTokens(),
			ExternalSkillSearcher:           configuredExternalSkillSearcher(),
			Logger:                          log.New(os.Stderr, "", 0),
		},
		HTTPServer: httpserver.HTTPServerConfig{
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

func envBoolOrDefault(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		log.Fatalf("parse %s: expected boolean value, got %q", key, value)
		return fallback
	}
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
