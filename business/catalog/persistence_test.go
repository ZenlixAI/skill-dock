package catalog

import (
	"context"
	"path/filepath"
	"testing"
)

func TestMemoryRepositoryPersistsAggregateSnapshot(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "catalog-state.json")

	repository := NewMemoryRepository(SampleData(), MemoryRepositoryOptions{
		AggregateStore: NewJSONAggregateStore(statePath),
	})

	if _, err := repository.CreateReleaseRecord(context.Background(), "@platform.skill/agent-notes", "0.1.0", ReleaseRecordInput{}); err != nil {
		t.Fatalf("create release record: %v", err)
	}
	if _, err := repository.ApplySkillGovernanceChange(context.Background(), "@platform.skill/code-review", SkillGovernanceChangeInput{
		Action: GovernanceActionRetire,
		Reason: "merged into successor",
	}, "user_123"); err != nil {
		t.Fatalf("apply skill governance: %v", err)
	}

	reloaded := NewMemoryRepository(SampleData(), MemoryRepositoryOptions{
		AggregateStore: NewJSONAggregateStore(statePath),
	})

	createdSkill, err := reloaded.GetSkill(context.Background(), "@platform.skill/agent-notes")
	if err != nil {
		t.Fatalf("get persisted skill: %v", err)
	}
	if createdSkill.DisplayName != "Agent Notes" {
		t.Fatalf("unexpected display name: %s", createdSkill.DisplayName)
	}

	if _, err := reloaded.ResolveInstallTarget(context.Background(), "@platform.skill/code-review", ""); err == nil {
		t.Fatal("expected retired skill to fail install resolution")
	} else {
		catalogErr, ok := err.(*Error)
		if !ok || catalogErr.Code != "skill_retired" {
			t.Fatalf("unexpected resolve error: %#v", err)
		}
	}
}
