package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"skill_registry/internal/catalog/domain"
	"skill_registry/internal/catalog/infra"
)

func TestCreateReleaseRecordIsIdempotentWhenStoredTimestampLostPrecision(t *testing.T) {
	const (
		packageName = "@platform.skill/runtime-basic"
		version     = "0.1.0"
	)

	publishedAt := time.Date(2026, time.July, 28, 9, 55, 3, 607000000, time.UTC)
	artifactVersion := domain.SkillVersionDetail{
		SkillVersionSummary: domain.SkillVersionSummary{
			PackageName:   packageName,
			Version:       version,
			VersionStatus: domain.VersionStatusPublished,
			InstallPolicy: domain.InstallPolicyResolvable,
			PublishedAt:   publishedAt,
		},
		Registry: domain.RegistryMetadata{
			NPMRegistry: "https://registry.example.com/skills-npm",
			TarballURL:  "https://registry.example.com/skills-npm/@platform.skill/runtime-basic/-/runtime-basic-0.1.0.tgz",
			Integrity:   "sha512-test",
			Shasum:      "test",
			PublishedAt: publishedAt,
			DistTags:    []string{"latest"},
		},
		Bundle: domain.BundleMetadata{
			SkillManifestDigest: "sha256:test",
			Components:          []string{"commands"},
		},
	}
	storedVersion := artifactVersion
	storedVersion.PublishedAt = publishedAt.Truncate(time.Second)
	storedVersion.Registry.PublishedAt = publishedAt.Truncate(time.Second)

	data := domain.SampleCatalogData{
		Skills: []domain.SkillRecord{{
			Detail: domain.SkillDetail{SkillSummary: domain.SkillSummary{
				PackageName: packageName,
				DisplayName: "Runtime Basic",
				Description: "Runtime fixture",
				SourceType:  domain.SourceTypePlatformCreated,
				Status:      domain.SkillStatusActive,
			}},
			Versions: []domain.SkillVersionDetail{storedVersion},
		}},
		Artifacts: []domain.ReleaseArtifact{{
			SkillDisplayName: "Runtime Basic",
			SkillDescription: "Runtime fixture",
			SkillSourceType:  domain.SourceTypePlatformCreated,
			Version:          artifactVersion,
		}},
	}
	repository := infra.NewMemoryRepository(data)

	result, err := repository.CreateReleaseRecord(context.Background(), packageName, version, domain.ReleaseRecordInput{})
	if err != nil {
		t.Fatalf("replay release record: %v", err)
	}
	if result.Created || !result.IdempotentReplay {
		t.Fatalf("expected idempotent replay, got %+v", result)
	}

	conflictingVersion := artifactVersion
	conflictingVersion.Bundle.SkillManifestDigest = "sha256:conflict"
	data.Artifacts[0].Version = conflictingVersion
	repository = infra.NewMemoryRepository(data)
	_, err = repository.CreateReleaseRecord(context.Background(), packageName, version, domain.ReleaseRecordInput{})
	var catalogErr *domain.Error
	if !errors.As(err, &catalogErr) || catalogErr.Code != "immutable_release_conflict" {
		t.Fatalf("expected immutable release conflict, got %v", err)
	}
}
