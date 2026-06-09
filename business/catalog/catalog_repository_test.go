package catalog

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestCatalogRepositoryCreateReleaseRecordPersistsThroughStateStore(t *testing.T) {
	store := NewMemoryAggregateStateStore(SampleData(), MemoryAggregateStateStoreOptions{})
	repo := NewCatalogRepository(CatalogRepositoryConfig{
		StateStore:     store,
		ArtifactSource: NewStaticArtifactSource(SampleData().Artifacts),
	})

	result, err := repo.CreateReleaseRecord(context.Background(), "@platform.skill/agent-notes", "0.1.0", ReleaseRecordInput{})
	if err != nil {
		t.Fatalf("create release record: %v", err)
	}
	if !result.Created {
		t.Fatalf("expected created release record, got %+v", result)
	}

	aggregate, err := store.GetAggregate(context.Background(), "@platform.skill/agent-notes")
	if err != nil {
		t.Fatalf("load aggregate: %v", err)
	}
	if aggregate.Revision == 0 {
		t.Fatalf("expected revision to bump, got %+v", aggregate)
	}
	if aggregate.Skill.Detail.PackageName != "@platform.skill/agent-notes" {
		t.Fatalf("unexpected aggregate package: %s", aggregate.Skill.Detail.PackageName)
	}
}

func TestCatalogRepositoryApplyVersionGovernanceChangeUpdatesAggregateLifecycle(t *testing.T) {
	store := NewMemoryAggregateStateStore(SampleData(), MemoryAggregateStateStoreOptions{})
	repo := NewCatalogRepository(CatalogRepositoryConfig{
		StateStore:     store,
		ArtifactSource: NewStaticArtifactSource(SampleData().Artifacts),
	})

	response, err := repo.ApplyVersionGovernanceChange(context.Background(), "@platform.skill/pdf-skill", "1.2.0", VersionGovernanceChangeInput{
		Action: GovernanceActionYank,
		Reason: "security issue",
	}, "user_123")
	if err != nil {
		t.Fatalf("apply governance change: %v", err)
	}
	if response.ResultingState.VersionStatus == nil || *response.ResultingState.VersionStatus != VersionStatusYanked {
		t.Fatalf("unexpected resulting state: %+v", response.ResultingState)
	}

	aggregate, err := store.GetAggregate(context.Background(), "@platform.skill/pdf-skill")
	if err != nil {
		t.Fatalf("load aggregate: %v", err)
	}
	if aggregate.Revision <= 1 {
		t.Fatalf("expected revision to increment, got %+v", aggregate)
	}

	found := false
	for _, version := range aggregate.Skill.Versions {
		if version.Version != "1.2.0" {
			continue
		}
		found = true
		if version.VersionStatus != VersionStatusYanked {
			t.Fatalf("expected version to be yanked, got %s", version.VersionStatus)
		}
		if version.InstallPolicy != InstallPolicyBlocked {
			t.Fatalf("expected install policy to be blocked, got %s", version.InstallPolicy)
		}
	}
	if !found {
		t.Fatal("expected governed version to remain in aggregate")
	}
}

func TestCatalogRepositoryListVersionsUsesConsistentQueryStoreOrdering(t *testing.T) {
	publishedLater := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	publishedEarlier := publishedLater.Add(-time.Hour)
	data := SampleCatalogData{
		Skills: []skillRecord{
			{
				Detail: SkillDetail{
					SkillSummary: SkillSummary{
						PackageName:         "@platform.skill/version-order-contract",
						DisplayName:         "Version Order Contract",
						Description:         "Locks query-store version ordering",
						SourceType:          SourceTypeImported,
						Status:              SkillStatusActive,
						LatestVersion:       stringPtr("1.10.0"),
						LatestVersionStatus: stringPtr(string(VersionStatusPublished)),
						UpdatedAt:           publishedLater,
					},
					CreatedAt: publishedEarlier.Add(-time.Hour),
				},
				Versions: []SkillVersionDetail{
					newMaterializedVersion("https://npm.internal.example.com", "@platform.skill/version-order-contract", "1.10.0", VersionStatusPublished, InstallPolicyResolvable, publishedEarlier, "Version Order Contract", "Locks query-store version ordering", []string{"commands"}),
					newMaterializedVersion("https://npm.internal.example.com", "@platform.skill/version-order-contract", "1.9.0", VersionStatusPublished, InstallPolicyResolvable, publishedLater, "Version Order Contract", "Locks query-store version ordering", []string{"commands"}),
					newMaterializedVersion("https://npm.internal.example.com", "@platform.skill/version-order-contract", "1.8.0", VersionStatusPublished, InstallPolicyResolvable, publishedEarlier.Add(-time.Hour), "Version Order Contract", "Locks query-store version ordering", []string{"commands"}),
				},
			},
		},
	}

	memoryRepo := NewCatalogRepository(CatalogRepositoryConfig{
		StateStore:     NewMemoryAggregateStateStore(data, MemoryAggregateStateStoreOptions{}),
		ArtifactSource: NewStaticArtifactSource(data.Artifacts),
	})
	sqlRepo := newSQLRepositoryForTest(t, data)

	query := VersionQuery{
		PackageName: "@platform.skill/version-order-contract",
		PageSize:    2,
	}
	memoryPage, err := memoryRepo.ListVersions(context.Background(), query)
	if err != nil {
		t.Fatalf("list memory versions: %v", err)
	}
	sqlPage, err := sqlRepo.ListVersions(context.Background(), query)
	if err != nil {
		t.Fatalf("list sql versions: %v", err)
	}

	memoryVersions := versionPageVersions(memoryPage)
	sqlVersions := versionPageVersions(sqlPage)
	want := []string{"1.9.0", "1.10.0"}
	if !reflect.DeepEqual(memoryVersions, want) {
		t.Fatalf("expected memory versions %v, got %v", want, memoryVersions)
	}
	if !reflect.DeepEqual(sqlVersions, want) {
		t.Fatalf("expected sql versions %v, got %v", want, sqlVersions)
	}
	if !reflect.DeepEqual(memoryVersions, sqlVersions) {
		t.Fatalf("expected backends to agree, memory=%v sql=%v", memoryVersions, sqlVersions)
	}
}

func TestSortSkillsBreaksUpdatedAtTiesByPackageName(t *testing.T) {
	tiedAt := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	items := []SkillSummary{
		{PackageName: "@platform.skill/zeta", UpdatedAt: tiedAt},
		{PackageName: "@platform.skill/alpha", UpdatedAt: tiedAt},
		{PackageName: "@platform.skill/mid", UpdatedAt: tiedAt},
	}

	sortSkills(items, "updated_at_desc", "")

	got := skillPagePackageNames(SkillsPage{Items: items})
	want := []string{"@platform.skill/alpha", "@platform.skill/mid", "@platform.skill/zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected tied updated_at rows ordered by package name, got %v", got)
	}
}

func TestCatalogRepositoryListSkillsUsesConsistentQueryStoreOrdering(t *testing.T) {
	tiedAt := time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)
	data := SampleCatalogData{
		Skills: []skillRecord{
			{
				Detail: SkillDetail{
					SkillSummary: SkillSummary{
						PackageName:         "@platform.skill/zeta",
						DisplayName:         "Tie Zeta",
						Description:         "Locks tied skill ordering",
						SourceType:          SourceTypeImported,
						Status:              SkillStatusActive,
						LatestVersion:       stringPtr("1.0.0"),
						LatestVersionStatus: stringPtr(string(VersionStatusPublished)),
						UpdatedAt:           tiedAt,
					},
					CreatedAt: tiedAt.Add(-time.Hour),
				},
				Versions: []SkillVersionDetail{
					newMaterializedVersion("https://npm.internal.example.com", "@platform.skill/zeta", "1.0.0", VersionStatusPublished, InstallPolicyResolvable, tiedAt, "Tie Zeta", "Locks tied skill ordering", []string{"commands"}),
				},
			},
			{
				Detail: SkillDetail{
					SkillSummary: SkillSummary{
						PackageName:         "@platform.skill/alpha",
						DisplayName:         "Tie Alpha",
						Description:         "Locks tied skill ordering",
						SourceType:          SourceTypeImported,
						Status:              SkillStatusActive,
						LatestVersion:       stringPtr("1.0.0"),
						LatestVersionStatus: stringPtr(string(VersionStatusPublished)),
						UpdatedAt:           tiedAt,
					},
					CreatedAt: tiedAt.Add(-time.Hour),
				},
				Versions: []SkillVersionDetail{
					newMaterializedVersion("https://npm.internal.example.com", "@platform.skill/alpha", "1.0.0", VersionStatusPublished, InstallPolicyResolvable, tiedAt, "Tie Alpha", "Locks tied skill ordering", []string{"commands"}),
				},
			},
			{
				Detail: SkillDetail{
					SkillSummary: SkillSummary{
						PackageName:         "@platform.skill/mid",
						DisplayName:         "Tie Mid",
						Description:         "Locks tied skill ordering",
						SourceType:          SourceTypeImported,
						Status:              SkillStatusActive,
						LatestVersion:       stringPtr("1.0.0"),
						LatestVersionStatus: stringPtr(string(VersionStatusPublished)),
						UpdatedAt:           tiedAt,
					},
					CreatedAt: tiedAt.Add(-time.Hour),
				},
				Versions: []SkillVersionDetail{
					newMaterializedVersion("https://npm.internal.example.com", "@platform.skill/mid", "1.0.0", VersionStatusPublished, InstallPolicyResolvable, tiedAt, "Tie Mid", "Locks tied skill ordering", []string{"commands"}),
				},
			},
		},
	}

	memoryRepo := NewCatalogRepository(CatalogRepositoryConfig{
		StateStore:     NewMemoryAggregateStateStore(data, MemoryAggregateStateStoreOptions{}),
		ArtifactSource: NewStaticArtifactSource(data.Artifacts),
	})
	sqlRepo := newSQLRepositoryForTest(t, data)

	query := SkillQuery{
		Sort:     "updated_at_desc",
		PageSize: 3,
	}
	sqlPage, err := sqlRepo.ListSkills(context.Background(), query)
	if err != nil {
		t.Fatalf("list sql skills: %v", err)
	}
	want := []string{"@platform.skill/alpha", "@platform.skill/mid", "@platform.skill/zeta"}
	if got := skillPagePackageNames(sqlPage); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected sql skills %v, got %v", want, got)
	}

	for i := 0; i < 20; i++ {
		memoryPage, err := memoryRepo.ListSkills(context.Background(), query)
		if err != nil {
			t.Fatalf("list memory skills: %v", err)
		}
		if got := skillPagePackageNames(memoryPage); !reflect.DeepEqual(got, want) {
			t.Fatalf("expected memory skills %v, got %v", want, got)
		}
	}
}

func versionPageVersions(page VersionsPage) []string {
	versions := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		versions = append(versions, item.Version)
	}
	return versions
}

func skillPagePackageNames(page SkillsPage) []string {
	names := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		names = append(names, item.PackageName)
	}
	return names
}
