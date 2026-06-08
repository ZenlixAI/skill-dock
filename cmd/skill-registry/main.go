package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi"
	"skill_registry/business/catalog"
	"skill_registry/business/packaging"
	artifactadapter "skill_registry/foundation/artifact"
	"skill_registry/foundation/persist"
)

func main() {
	addr := envOrDefault("SKILL_REGISTRY_ADDR", ":8080")
	enableSampleData := envBoolOrDefault("SKILL_REGISTRY_ENABLE_SAMPLE_DATA", false)
	enableMockRegistry := envBoolOrDefault("SKILL_REGISTRY_ENABLE_MOCK_REGISTRY", false)
	sampleData := configuredSampleData(enableSampleData)

	artifactSource := configuredArtifactSource(sampleData, enableSampleData)
	databaseURL := envOrDefault("SKILL_REGISTRY_DATABASE_URL", "file:"+filepath.Join("data", "skill-registry.db"))
	repository, err := catalog.NewSQLRepository(context.Background(), databaseURL, sampleData, catalog.SQLRepositoryOptions{
		ArtifactSource: artifactSource,
		DisableSeed:    !enableSampleData,
	})
	if err != nil {
		log.Fatal(err)
	}
	storeSet, err := persist.NewSQLStoreSet(context.Background(), databaseURL, 10*time.Minute)
	if err != nil {
		log.Fatal(err)
	}

	server := catalogapi.NewServer(catalogapi.ServerConfig{
		ReaderToken:                     os.Getenv("SKILL_REGISTRY_READER_TOKEN"),
		AuthoringToken:                  os.Getenv("SKILL_REGISTRY_AUTHORING_TOKEN"),
		PublisherToken:                  os.Getenv("SKILL_REGISTRY_PUBLISHER_TOKEN"),
		NPMRegistryURL:                  os.Getenv("SKILL_REGISTRY_NPM_REGISTRY_URL"),
		NPMRegistryPublishAuthorization: configuredNPMPublishAuthorization(),
		PublicNPMRegistryURL:            os.Getenv("SKILL_REGISTRY_PUBLIC_NPM_REGISTRY_URL"),
		AuthTokens:                      configuredAuthTokens(),
		SessionSecret:                   os.Getenv("SKILL_REGISTRY_SESSION_SECRET"),
		SessionUsers:                    configuredSessionUsers(),
		MockRegistry:                    configuredMockRegistry(sampleData, enableMockRegistry),
		DisableMockRegistry:             !enableMockRegistry,
		Packaging:                       packaging.NewServiceWithWorkspaceStore("", storeSet.Workspace),
		Repository:                      repository,
		Idempotency:                     storeSet.Idempotency,
		Audit:                           storeSet.Audit,
		Logger:                          log.New(os.Stderr, "", 0),
	})

	log.Printf("skill registry listening on %s", addr)
	if err := http.ListenAndServe(addr, server.Handler()); err != nil {
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

func defaultArtifactSource(sampleData catalog.SampleCatalogData) catalog.ArtifactSource {
	defaultPath := filepath.Join("data", "registry-artifacts.sample.json")
	if _, err := os.Stat(defaultPath); err == nil {
		return artifactadapter.NewJSONArtifactSource(defaultPath)
	}
	return catalog.NewStaticArtifactSource(sampleData.Artifacts)
}

func configuredArtifactSource(sampleData catalog.SampleCatalogData, enableSampleData bool) catalog.ArtifactSource {
	if path := os.Getenv("SKILL_REGISTRY_ARTIFACTS_PATH"); path != "" {
		return artifactadapter.NewJSONArtifactSource(path)
	}
	if registryURL := os.Getenv("SKILL_REGISTRY_NPM_REGISTRY_URL"); registryURL != "" {
		return artifactadapter.NewNPMRegistryArtifactSource(registryURL, nil)
	}
	if !enableSampleData {
		return catalog.NewStaticArtifactSource(nil)
	}
	return defaultArtifactSource(sampleData)
}

func configuredSampleData(enableSampleData bool) catalog.SampleCatalogData {
	if !enableSampleData {
		return catalog.SampleCatalogData{}
	}
	sampleRegistryBase := envOrDefault("SKILL_REGISTRY_SAMPLE_NPM_BASE_URL", "https://npm.internal.example.com")
	return catalog.SampleDataWithRegistryBase(sampleRegistryBase)
}

func configuredMockRegistry(sampleData catalog.SampleCatalogData, enableMockRegistry bool) *artifactadapter.MockRegistry {
	if !enableMockRegistry {
		return nil
	}
	return artifactadapter.NewMockRegistry(sampleData.Artifacts)
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

func envBoolOrDefault(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "TRUE", "True", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "False", "no", "NO", "off", "OFF":
		return false
	default:
		log.Fatalf("parse %s: expected boolean value, got %q", key, value)
		return fallback
	}
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
