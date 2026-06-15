package catalog

import (
	"context"
	"errors"

	internalapp "skill_registry/internal/catalog/application"
	internaldomain "skill_registry/internal/catalog/domain"
)

type WriteModule struct {
	service *internalapp.WriteService
}

func newWriteModule(stateStore AggregateStateStore, artifacts ArtifactSource) *WriteModule {
	var artifactAdapter *artifactSourceAdapter
	if artifacts != nil {
		artifactAdapter = &artifactSourceAdapter{source: artifacts}
	}
	return &WriteModule{
		service: internalapp.NewWriteService(
			&aggregateStateStoreAdapter{store: stateStore},
			artifactAdapter,
		),
	}
}

func (m *WriteModule) Ping(ctx context.Context) error {
	return m.service.Ping(ctx)
}

func (m *WriteModule) Seed(ctx context.Context, data SampleCatalogData) error {
	return m.service.Seed(ctx, data)
}

func (m *WriteModule) CreateReleaseRecord(ctx context.Context, packageName string, version string, input ReleaseRecordInput) (ReleaseRecordResult, error) {
	return m.service.CreateReleaseRecord(ctx, packageName, version, input)
}

func (m *WriteModule) ApplySkillGovernanceChange(ctx context.Context, packageName string, input SkillGovernanceChangeInput, actor string) (GovernanceChangeResponse, error) {
	return m.service.ApplySkillGovernanceChange(ctx, packageName, input, actor)
}

func (m *WriteModule) ApplyVersionGovernanceChange(ctx context.Context, packageName string, version string, input VersionGovernanceChangeInput, actor string) (GovernanceChangeResponse, error) {
	return m.service.ApplyVersionGovernanceChange(ctx, packageName, version, input, actor)
}

type artifactSourceAdapter struct {
	source ArtifactSource
}

func (a *artifactSourceAdapter) FindReleaseArtifact(ctx context.Context, packageName string, version string) (internaldomain.ReleaseArtifact, error) {
	if a == nil || a.source == nil {
		return internaldomain.ReleaseArtifact{}, nil
	}
	return a.source.FindReleaseArtifact(ctx, packageName, version)
}

type aggregateStateStoreAdapter struct {
	store AggregateStateStore
}

func (a *aggregateStateStoreAdapter) IsEmpty(ctx context.Context) (bool, error) {
	return a.store.IsEmpty(ctx)
}

func (a *aggregateStateStoreAdapter) GetAggregate(ctx context.Context, packageName string) (internaldomain.AgentSkillAggregate, error) {
	aggregate, err := a.store.GetAggregate(ctx, packageName)
	if errors.Is(err, ErrAgentSkillAggregateNotFound) {
		return internaldomain.AgentSkillAggregate{}, internaldomain.ErrAgentSkillAggregateNotFound
	}
	if err != nil {
		return internaldomain.AgentSkillAggregate{}, err
	}
	return toInternalAggregate(aggregate), nil
}

func (a *aggregateStateStoreAdapter) PutAggregate(ctx context.Context, aggregate internaldomain.AgentSkillAggregate, expectedRevision uint64) error {
	if err := a.store.PutAggregate(ctx, fromInternalAggregate(aggregate), expectedRevision); err != nil {
		if errors.Is(err, ErrAggregateRevisionConflict) {
			return internaldomain.ErrAggregateRevisionConflict
		}
		return err
	}
	return nil
}

func (a *aggregateStateStoreAdapter) Ping(ctx context.Context) error {
	return a.store.Ping(ctx)
}

func toInternalAggregate(aggregate AgentSkillAggregate) internaldomain.AgentSkillAggregate {
	return internaldomain.AgentSkillAggregate{
		Revision: aggregate.Revision,
		Skill: internaldomain.SkillSnapshot{
			Detail:         aggregate.Skill.Detail,
			CurationReason: aggregate.Skill.CurationReason,
			Versions:       append([]internaldomain.SkillVersionDetail(nil), aggregate.Skill.Versions...),
		},
		Governance: append([]internaldomain.GovernanceRecord(nil), aggregate.Governance...),
	}
}

func fromInternalAggregate(aggregate internaldomain.AgentSkillAggregate) AgentSkillAggregate {
	return AgentSkillAggregate{
		Revision: aggregate.Revision,
		Skill: SkillSnapshot{
			Detail:         aggregate.Skill.Detail,
			CurationReason: aggregate.Skill.CurationReason,
			Versions:       append([]SkillVersionDetail(nil), aggregate.Skill.Versions...),
		},
		Governance: append([]GovernanceRecord(nil), aggregate.Governance...),
	}
}
