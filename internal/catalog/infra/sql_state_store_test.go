package infra

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"
	"time"

	. "skill_registry/internal/catalog/domain"
)

func TestSQLStateStorePreservesVersionPublishTimestampPrecision(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer database.Close()

	store := NewSQLStateStore(database, "sqlite")
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	publishedAt := time.Date(2026, time.July, 28, 9, 55, 3, 607000000, time.UTC)
	aggregate := AgentSkillAggregate{Skill: SkillSnapshot{
		Detail: SkillDetail{SkillSummary: SkillSummary{
			PackageName: "@platform.skill/runtime-basic",
			DisplayName: "Runtime Basic",
			Description: "Runtime fixture",
			SourceType:  SourceTypePlatformCreated,
			Status:      SkillStatusActive,
			UpdatedAt:   publishedAt,
		}, CreatedAt: publishedAt},
		Versions: []SkillVersionDetail{{
			SkillVersionSummary: SkillVersionSummary{
				PackageName:   "@platform.skill/runtime-basic",
				Version:       "0.1.0",
				VersionStatus: VersionStatusPublished,
				InstallPolicy: InstallPolicyResolvable,
				PublishedAt:   publishedAt,
			},
			Registry: RegistryMetadata{
				NPMRegistry: "https://registry.example.com/skills-npm",
				TarballURL:  "https://registry.example.com/runtime-basic-0.1.0.tgz",
				Integrity:   "sha512-test",
				Shasum:      "test",
				PublishedAt: publishedAt,
			},
			Bundle: BundleMetadata{SkillManifestDigest: "sha256:test"},
		}},
	}}
	if err := store.PutAggregate(ctx, aggregate, 0); err != nil {
		t.Fatalf("store aggregate: %v", err)
	}

	loaded, err := store.GetAggregate(ctx, "@platform.skill/runtime-basic")
	if err != nil {
		t.Fatalf("load aggregate: %v", err)
	}
	if got := loaded.Skill.Versions[0].PublishedAt; !got.Equal(publishedAt) {
		t.Fatalf("expected publishedAt %s, got %s", publishedAt, got)
	}
}

func TestSQLStateStoreListSkillsSQLAppliesSortLimitOffset(t *testing.T) {
	store := &SQLStateStore{dialect: "sqlite"}
	query := SkillQuery{
		Sort:     "package_name_asc",
		PageSize: 1,
		Cursor:   1,
	}

	sortKey, err := normalizedSkillsSort(query.Sort, query.Query)
	if err != nil {
		t.Fatalf("normalize sort: %v", err)
	}

	sqlText := store.listSkillsSQL(query, sortKey)
	if !strings.Contains(sqlText, "ORDER BY package_name ASC") {
		t.Fatalf("expected package sort in SQL, got %q", sqlText)
	}
	if !strings.Contains(sqlText, "LIMIT ?") {
		t.Fatalf("expected SQL limit clause, got %q", sqlText)
	}
	if !strings.Contains(sqlText, "OFFSET ?") {
		t.Fatalf("expected SQL offset clause, got %q", sqlText)
	}

	args := store.listSkillsArgs(query, sortKey)
	want := []any{2, 1}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("expected limit/offset args %v, got %v", want, args)
	}
}
