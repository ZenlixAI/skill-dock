package artifact

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	catalog "skill_registry/internal/catalog/domain"
)

type JSONArtifactSource struct {
	path string
}

func NewJSONArtifactSource(path string) *JSONArtifactSource {
	return &JSONArtifactSource{path: path}
}

func (s *JSONArtifactSource) FindReleaseArtifact(_ context.Context, packageName string, version string) (catalog.ReleaseArtifact, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return catalog.ReleaseArtifact{}, catalog.NewError(404, "registry_version_not_found", "Registry version not found.", map[string]any{
				"packageName": packageName,
				"version":     version,
			})
		}
		return catalog.ReleaseArtifact{}, catalog.NewError(503, "registry_unavailable", "Registry is unavailable.", map[string]any{
			"packageName": packageName,
			"version":     version,
			"path":        s.path,
		})
	}

	var artifacts []catalog.ReleaseArtifact
	if err := json.Unmarshal(content, &artifacts); err != nil {
		return catalog.ReleaseArtifact{}, catalog.NewError(503, "registry_unavailable", "Registry metadata is unreadable.", map[string]any{
			"path": s.path,
		})
	}

	for _, artifact := range artifacts {
		if artifact.Version.PackageName == packageName && artifact.Version.Version == version {
			return artifact, nil
		}
	}

	return catalog.ReleaseArtifact{}, catalog.NewError(404, "registry_version_not_found", "Registry version not found.", map[string]any{
		"packageName": packageName,
		"version":     version,
	})
}

type NPMRegistryArtifactSource struct {
	baseURL           string
	client            *http.Client
	defaultSourceType catalog.SourceType
}

func NewNPMRegistryArtifactSource(baseURL string, client *http.Client) *NPMRegistryArtifactSource {
	if client == nil {
		client = http.DefaultClient
	}
	return &NPMRegistryArtifactSource{
		baseURL:           strings.TrimRight(baseURL, "/"),
		client:            client,
		defaultSourceType: catalog.SourceTypeImported,
	}
}

func (s *NPMRegistryArtifactSource) FindReleaseArtifact(ctx context.Context, packageName string, version string) (catalog.ReleaseArtifact, error) {
	if strings.TrimSpace(s.baseURL) == "" {
		return catalog.ReleaseArtifact{}, catalog.NewError(503, "registry_unavailable", "Registry is unavailable.", map[string]any{
			"packageName": packageName,
			"version":     version,
		})
	}

	requestURL := s.baseURL + "/" + url.PathEscape(packageName)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return catalog.ReleaseArtifact{}, catalog.NewError(503, "registry_unavailable", "Registry request could not be created.", map[string]any{
			"packageName": packageName,
			"version":     version,
		})
	}

	response, err := s.client.Do(request)
	if err != nil {
		return catalog.ReleaseArtifact{}, catalog.NewError(503, "registry_unavailable", "Registry is unavailable.", map[string]any{
			"packageName": packageName,
			"version":     version,
		})
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return catalog.ReleaseArtifact{}, catalog.NewError(404, "registry_version_not_found", "Registry version not found.", map[string]any{
			"packageName": packageName,
			"version":     version,
		})
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return catalog.ReleaseArtifact{}, catalog.NewError(503, "registry_unavailable", "Registry is unavailable.", map[string]any{
			"packageName": packageName,
			"version":     version,
			"status":      response.StatusCode,
		})
	}

	var metadata npmPackageMetadata
	if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
		return catalog.ReleaseArtifact{}, catalog.NewError(503, "registry_unavailable", "Registry metadata is unreadable.", map[string]any{
			"packageName": packageName,
			"version":     version,
		})
	}

	versionMetadata, ok := metadata.Versions[version]
	if !ok {
		return catalog.ReleaseArtifact{}, catalog.NewError(404, "registry_version_not_found", "Registry version not found.", map[string]any{
			"packageName": packageName,
			"version":     version,
		})
	}

	publishedAt, err := parseRegistryPublishedAt(metadata.Time[version])
	if err != nil {
		return catalog.ReleaseArtifact{}, catalog.NewError(503, "registry_unavailable", "Registry metadata is unreadable.", map[string]any{
			"packageName": packageName,
			"version":     version,
			"field":       "time",
		})
	}

	artifact := catalog.ReleaseArtifact{
		SkillDisplayName: firstNonEmpty(versionMetadata.Skill.DisplayName, deriveDisplayName(packageName)),
		SkillDescription: firstNonEmpty(versionMetadata.Description, metadata.Description),
		SkillSourceType:  firstSourceType(versionMetadata.Skill.SourceType, s.defaultSourceType),
		Version: catalog.SkillVersionDetail{
			SkillVersionSummary: catalog.SkillVersionSummary{
				PackageName:   packageName,
				Version:       version,
				VersionStatus: catalog.VersionStatusPublished,
				InstallPolicy: catalog.InstallPolicyResolvable,
				PublishedAt:   publishedAt,
			},
			Registry: catalog.RegistryMetadata{
				NPMRegistry: s.baseURL,
				TarballURL:  versionMetadata.Dist.Tarball,
				Integrity:   versionMetadata.Dist.Integrity,
				Shasum:      versionMetadata.Dist.Shasum,
				PublishedAt: publishedAt,
				DistTags:    matchingDistTags(metadata.DistTags, version),
			},
		},
		ValidationError: versionMetadata.Skill.ValidationError,
	}

	if validationError := validateRegistryArtifactMetadata(packageName, version, versionMetadata, artifact); validationError != "" {
		artifact.ValidationError = validationError
		return artifact, nil
	}

	scanResult, validationError, err := s.scanTarball(ctx, artifact.Version.Registry.TarballURL, packageName, version)
	if err != nil {
		return catalog.ReleaseArtifact{}, err
	}
	if validationError != "" {
		artifact.ValidationError = validationError
		return artifact, nil
	}

	artifact.SkillDisplayName = firstNonEmpty(scanResult.DisplayName, artifact.SkillDisplayName)
	artifact.SkillDescription = firstNonEmpty(scanResult.Description, artifact.SkillDescription)
	artifact.Version.Bundle = catalog.BundleMetadata{
		SkillManifestDigest: scanResult.ManifestDigest,
		Components:          scanResult.Components,
	}
	return artifact, nil
}

type npmPackageMetadata struct {
	Name        string                        `json:"name"`
	Description string                        `json:"description"`
	DistTags    map[string]string             `json:"dist-tags"`
	Time        map[string]string             `json:"time"`
	Versions    map[string]npmVersionMetadata `json:"versions"`
}

type npmVersionMetadata struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Dist        npmDistMetadata        `json:"dist"`
	Skill       npmSkillBundleMetadata `json:"skill"`
}

type npmDistMetadata struct {
	Tarball   string `json:"tarball"`
	Integrity string `json:"integrity"`
	Shasum    string `json:"shasum"`
}

type npmSkillBundleMetadata struct {
	DisplayName     string             `json:"displayName"`
	SourceType      catalog.SourceType `json:"sourceType"`
	ManifestDigest  string             `json:"manifestDigest"`
	Components      []string           `json:"components"`
	ValidationError string             `json:"validationError"`
}

func parseRegistryPublishedAt(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, fmt.Errorf("published time missing")
	}
	return time.Parse(time.RFC3339, value)
}

func validateRegistryArtifactMetadata(packageName string, version string, metadata npmVersionMetadata, artifact catalog.ReleaseArtifact) string {
	switch {
	case metadata.Name != "" && metadata.Name != packageName:
		return "package metadata name does not match requested package"
	case metadata.Version != "" && metadata.Version != version:
		return "package metadata version does not match requested version"
	case strings.TrimSpace(artifact.Version.Registry.TarballURL) == "":
		return "dist.tarball is required"
	case strings.TrimSpace(artifact.Version.Registry.Integrity) == "":
		return "dist.integrity is required"
	case strings.TrimSpace(artifact.Version.Registry.Shasum) == "":
		return "dist.shasum is required"
	default:
		return ""
	}
}

func matchingDistTags(tags map[string]string, version string) []string {
	result := make([]string, 0, len(tags))
	for name, value := range tags {
		if value == version {
			result = append(result, name)
		}
	}
	return result
}

func deriveDisplayName(packageName string) string {
	parts := strings.Split(packageName, "/")
	base := parts[len(parts)-1]
	words := strings.Split(base, "-")
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstSourceType(value catalog.SourceType, fallback catalog.SourceType) catalog.SourceType {
	if value == "" {
		return fallback
	}
	return value
}

func (s *NPMRegistryArtifactSource) scanTarball(ctx context.Context, tarballURL string, packageName string, version string) (BundleScanResult, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, tarballURL, nil)
	if err != nil {
		return BundleScanResult{}, "", catalog.NewError(503, "registry_unavailable", "Registry tarball request could not be created.", map[string]any{
			"packageName": packageName,
			"version":     version,
		})
	}

	response, err := s.client.Do(request)
	if err != nil {
		return BundleScanResult{}, "", catalog.NewError(503, "registry_unavailable", "Registry is unavailable.", map[string]any{
			"packageName": packageName,
			"version":     version,
		})
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return BundleScanResult{}, "", catalog.NewError(503, "registry_unavailable", "Registry tarball is unavailable.", map[string]any{
			"packageName": packageName,
			"version":     version,
			"status":      response.StatusCode,
		})
	}

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return BundleScanResult{}, "", catalog.NewError(503, "registry_unavailable", "Registry tarball could not be read.", map[string]any{
			"packageName": packageName,
			"version":     version,
		})
	}

	scanResult, validationError := scanNPMPackageTarball(bytes.NewReader(content), packageName, version)
	return scanResult, validationError, nil
}
