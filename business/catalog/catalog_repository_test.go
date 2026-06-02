package catalog

import (
	"context"
	"testing"
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
