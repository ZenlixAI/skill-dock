package catalog

import (
	"context"
	"testing"
)

type writeModuleAggregateStateStoreStub struct {
	aggregates   map[string]AgentSkillAggregate
	getCalls     int
	putCalls     int
	lastExpected uint64
}

var _ AggregateStateStore = (*writeModuleAggregateStateStoreStub)(nil)

func newWriteModuleAggregateStateStoreStub(data SampleCatalogData) *writeModuleAggregateStateStoreStub {
	aggregates := make(map[string]AgentSkillAggregate)
	for _, aggregate := range aggregatesFromSampleData(data) {
		aggregates[aggregate.Skill.Detail.PackageName] = cloneAggregate(aggregate)
	}
	return &writeModuleAggregateStateStoreStub{aggregates: aggregates}
}

func (s *writeModuleAggregateStateStoreStub) IsEmpty(context.Context) (bool, error) {
	return len(s.aggregates) == 0, nil
}

func (s *writeModuleAggregateStateStoreStub) GetAggregate(_ context.Context, packageName string) (AgentSkillAggregate, error) {
	s.getCalls++
	aggregate, ok := s.aggregates[packageName]
	if !ok {
		return AgentSkillAggregate{}, ErrAgentSkillAggregateNotFound
	}
	return cloneAggregate(aggregate), nil
}

func (s *writeModuleAggregateStateStoreStub) PutAggregate(_ context.Context, aggregate AgentSkillAggregate, expectedRevision uint64) error {
	s.putCalls++
	s.lastExpected = expectedRevision

	current, ok := s.aggregates[aggregate.Skill.Detail.PackageName]
	switch {
	case ok && current.Revision != expectedRevision:
		return ErrAggregateRevisionConflict
	case !ok && expectedRevision != 0:
		return ErrAggregateRevisionConflict
	}

	updated := cloneAggregate(aggregate)
	updated.Revision = expectedRevision + 1
	s.aggregates[updated.Skill.Detail.PackageName] = updated
	return nil
}

func (s *writeModuleAggregateStateStoreStub) Ping(context.Context) error {
	return nil
}

func TestWriteModuleAppliesVersionGovernanceChangeThroughAggregateState(t *testing.T) {
	store := newWriteModuleAggregateStateStoreStub(SampleData())
	module := newWriteModule(store, NewStaticArtifactSource(SampleData().Artifacts))

	response, err := module.ApplyVersionGovernanceChange(context.Background(), "@platform.skill/pdf-skill", "1.2.0", VersionGovernanceChangeInput{
		Action: GovernanceActionYank,
		Reason: "security issue",
	}, "user_123")
	if err != nil {
		t.Fatalf("apply governance change: %v", err)
	}
	if response.ResultingState.VersionStatus == nil || *response.ResultingState.VersionStatus != VersionStatusYanked {
		t.Fatalf("unexpected resulting state: %+v", response.ResultingState)
	}
	if store.getCalls == 0 || store.putCalls == 0 {
		t.Fatalf("expected aggregate load/save calls, got get=%d put=%d", store.getCalls, store.putCalls)
	}

	aggregate := store.aggregates["@platform.skill/pdf-skill"]
	for _, version := range aggregate.Skill.Versions {
		if version.Version != "1.2.0" {
			continue
		}
		if version.VersionStatus != VersionStatusYanked {
			t.Fatalf("expected version to be yanked, got %s", version.VersionStatus)
		}
		if version.InstallPolicy != InstallPolicyBlocked {
			t.Fatalf("expected install policy to be blocked, got %s", version.InstallPolicy)
		}
		return
	}
	t.Fatal("expected governed version to remain in aggregate state")
}
