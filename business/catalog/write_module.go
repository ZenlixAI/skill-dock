package catalog

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

const maxAggregateWriteRetries = 5

type WriteModule struct {
	stateStore AggregateStateStore
	artifacts  ArtifactSource
}

func newWriteModule(stateStore AggregateStateStore, artifacts ArtifactSource) *WriteModule {
	return &WriteModule{
		stateStore: stateStore,
		artifacts:  artifacts,
	}
}

func (m *WriteModule) Ping(ctx context.Context) error {
	return m.stateStore.Ping(ctx)
}

func (m *WriteModule) Seed(ctx context.Context, data SampleCatalogData) error {
	if len(data.Skills) == 0 {
		return nil
	}
	empty, err := m.stateStore.IsEmpty(ctx)
	if err != nil {
		return err
	}
	if !empty {
		return nil
	}
	for _, aggregate := range aggregatesFromSampleData(data) {
		if err := m.stateStore.PutAggregate(ctx, aggregate, 0); err != nil {
			if errors.Is(err, ErrAggregateRevisionConflict) {
				return nil
			}
			return err
		}
	}
	return nil
}

func (m *WriteModule) CreateReleaseRecord(ctx context.Context, packageName string, version string, input ReleaseRecordInput) (ReleaseRecordResult, error) {
	if !isExactVersion(version) {
		return ReleaseRecordResult{}, NewError(422, "invalid_request", "version must be an exact semver string.", map[string]any{"version": version})
	}
	if m.artifacts == nil {
		return ReleaseRecordResult{}, NewError(500, "internal_error", "Artifact source is not configured.", nil)
	}

	artifact, err := m.artifacts.FindReleaseArtifact(ctx, packageName, version)
	if err != nil {
		return ReleaseRecordResult{}, err
	}
	if artifact.RegistryDown {
		return ReleaseRecordResult{}, NewError(503, "registry_unavailable", "Registry is unavailable.", map[string]any{
			"packageName": packageName,
			"version":     version,
		})
	}
	if artifact.ValidationError != "" {
		return ReleaseRecordResult{}, NewError(422, "bundle_validation_failed", "Bundle validation failed.", map[string]any{
			"packageName": packageName,
			"version":     version,
			"reason":      artifact.ValidationError,
		})
	}
	if validationError := validateReleaseRecordInput(input, artifact); validationError != nil {
		return ReleaseRecordResult{}, validationError
	}

	return updateAggregate(ctx, m.stateStore, packageName, true, func(aggregate AgentSkillAggregate, exists bool) (AgentSkillAggregate, ReleaseRecordResult, error) {
		record := aggregateSkillRecord(aggregate)
		if !exists {
			record = skillRecord{
				Detail: SkillDetail{
					SkillSummary: SkillSummary{
						PackageName: packageName,
						DisplayName: artifact.SkillDisplayName,
						Description: artifact.SkillDescription,
						SourceType:  artifact.SkillSourceType,
						Status:      SkillStatusActive,
					},
					CreatedAt: artifact.Version.Registry.PublishedAt,
				},
			}
		}

		for _, existing := range record.Versions {
			if existing.Version != version {
				continue
			}
			if reflect.DeepEqual(existing, artifact.Version) {
				return aggregate, ReleaseRecordResult{
					PackageName:      packageName,
					Version:          version,
					Created:          false,
					IdempotentReplay: true,
					VersionStatus:    existing.VersionStatus,
					InstallPolicy:    existing.InstallPolicy,
				}, nil
			}
			return AgentSkillAggregate{}, ReleaseRecordResult{}, NewError(409, "immutable_release_conflict", "Immutable release content conflicts with the existing record.", map[string]any{
				"packageName": packageName,
				"version":     version,
			})
		}

		record.Versions = append(record.Versions, artifact.Version)
		record.Detail.UpdatedAt = artifact.Version.Registry.PublishedAt
		refreshLatestVersion(&record)

		updated := aggregateFromSkillRecord(record)
		updated.Revision = aggregate.Revision
		updated.Governance = append([]GovernanceRecord(nil), aggregate.Governance...)
		return updated, ReleaseRecordResult{
			PackageName:      packageName,
			Version:          version,
			Created:          true,
			IdempotentReplay: false,
			VersionStatus:    artifact.Version.VersionStatus,
			InstallPolicy:    artifact.Version.InstallPolicy,
		}, nil
	})
}

func (m *WriteModule) ApplySkillGovernanceChange(ctx context.Context, packageName string, input SkillGovernanceChangeInput, actor string) (GovernanceChangeResponse, error) {
	if input.Action != GovernanceActionRetire {
		return GovernanceChangeResponse{}, NewError(422, "invalid_request", "skill governance action is invalid.", map[string]any{"action": input.Action})
	}
	if strings.TrimSpace(input.Reason) == "" {
		return GovernanceChangeResponse{}, NewError(422, "invalid_request", "reason is required.", map[string]any{"field": "reason"})
	}

	return updateAggregate(ctx, m.stateStore, packageName, false, func(aggregate AgentSkillAggregate, _ bool) (AgentSkillAggregate, GovernanceChangeResponse, error) {
		record := aggregateSkillRecord(aggregate)
		if record.Detail.Status == SkillStatusRetired {
			return AgentSkillAggregate{}, GovernanceChangeResponse{}, NewError(409, "invalid_state_transition", "Skill is already retired.", map[string]any{"packageName": packageName})
		}

		record.Detail.Status = SkillStatusRetired
		record.Detail.UpdatedAt = time.Now().UTC()

		governanceRecord := GovernanceRecord{
			ID:          nextGovernanceID(packageName, aggregate.Revision+1),
			TargetType:  GovernanceTargetSkill,
			PackageName: packageName,
			Action:      GovernanceActionRetire,
			Reason:      input.Reason,
			ChangedBy:   defaultActor(actor),
			ChangedAt:   record.Detail.UpdatedAt,
		}

		updated := aggregateFromSkillRecord(record)
		updated.Revision = aggregate.Revision
		updated.Governance = append(append([]GovernanceRecord(nil), aggregate.Governance...), governanceRecord)

		status := SkillStatusRetired
		return updated, GovernanceChangeResponse{
			Record: governanceRecord,
			ResultingState: GovernanceChangeResultingState{
				Status: &status,
			},
		}, nil
	})
}

func (m *WriteModule) ApplyVersionGovernanceChange(ctx context.Context, packageName string, version string, input VersionGovernanceChangeInput, actor string) (GovernanceChangeResponse, error) {
	if !isExactVersion(version) {
		return GovernanceChangeResponse{}, NewError(422, "invalid_request", "version must be an exact semver string.", map[string]any{"version": version})
	}
	if input.Action != GovernanceActionDeprecate && input.Action != GovernanceActionYank {
		return GovernanceChangeResponse{}, NewError(422, "invalid_request", "version governance action is invalid.", map[string]any{"action": input.Action})
	}
	if strings.TrimSpace(input.Reason) == "" {
		return GovernanceChangeResponse{}, NewError(422, "invalid_request", "reason is required.", map[string]any{"field": "reason"})
	}

	return updateAggregate(ctx, m.stateStore, packageName, false, func(aggregate AgentSkillAggregate, _ bool) (AgentSkillAggregate, GovernanceChangeResponse, error) {
		record := aggregateSkillRecord(aggregate)

		index := -1
		for i, item := range record.Versions {
			if item.Version == version {
				index = i
				break
			}
		}
		if index < 0 {
			return AgentSkillAggregate{}, GovernanceChangeResponse{}, NewError(404, "version_not_found", "Version not found.", map[string]any{
				"packageName": packageName,
				"version":     version,
			})
		}

		current := record.Versions[index]
		switch input.Action {
		case GovernanceActionDeprecate:
			if current.VersionStatus != VersionStatusPublished {
				return AgentSkillAggregate{}, GovernanceChangeResponse{}, NewError(409, "invalid_state_transition", "Only published versions can be deprecated.", map[string]any{
					"packageName": packageName,
					"version":     version,
				})
			}
			current.VersionStatus = VersionStatusDeprecated
		case GovernanceActionYank:
			if current.VersionStatus == VersionStatusYanked {
				return AgentSkillAggregate{}, GovernanceChangeResponse{}, NewError(409, "invalid_state_transition", "Version is already yanked.", map[string]any{
					"packageName": packageName,
					"version":     version,
				})
			}
			current.VersionStatus = VersionStatusYanked
			current.InstallPolicy = InstallPolicyBlocked
		}

		record.Versions[index] = current
		record.Detail.UpdatedAt = time.Now().UTC()
		refreshLatestVersion(&record)

		governanceRecord := GovernanceRecord{
			ID:                        nextGovernanceID(packageName, aggregate.Revision+1),
			TargetType:                GovernanceTargetVersion,
			PackageName:               packageName,
			Version:                   stringPtr(version),
			Action:                    input.Action,
			Reason:                    input.Reason,
			ReplacementRecommendation: input.ReplacementRecommendation,
			ChangedBy:                 defaultActor(actor),
			ChangedAt:                 record.Detail.UpdatedAt,
		}

		updated := aggregateFromSkillRecord(record)
		updated.Revision = aggregate.Revision
		updated.Governance = append(append([]GovernanceRecord(nil), aggregate.Governance...), governanceRecord)

		resultingStatus := current.VersionStatus
		return updated, GovernanceChangeResponse{
			Record: governanceRecord,
			ResultingState: GovernanceChangeResultingState{
				VersionStatus: &resultingStatus,
			},
		}, nil
	})
}

func aggregateSkillRecord(aggregate AgentSkillAggregate) skillRecord {
	return skillRecord{
		Detail:         aggregate.Skill.Detail,
		CurationReason: aggregate.Skill.CurationReason,
		Versions:       append([]SkillVersionDetail(nil), aggregate.Skill.Versions...),
	}
}

func aggregateFromSkillRecord(record skillRecord) AgentSkillAggregate {
	return AgentSkillAggregate{
		Skill: SkillSnapshot{
			Detail:         record.Detail,
			CurationReason: record.CurationReason,
			Versions:       append([]SkillVersionDetail(nil), record.Versions...),
		},
	}
}

func nextGovernanceID(packageName string, revision uint64) string {
	var builder strings.Builder
	for _, char := range packageName {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
		case char >= 'A' && char <= 'Z':
			builder.WriteRune(char + ('a' - 'A'))
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
		default:
			builder.WriteByte('_')
		}
	}
	return fmt.Sprintf("gov_%s_%06d", builder.String(), revision)
}

func updateAggregate[T any](
	ctx context.Context,
	store AggregateStateStore,
	packageName string,
	allowCreate bool,
	mutate func(aggregate AgentSkillAggregate, exists bool) (AgentSkillAggregate, T, error),
) (T, error) {
	var zero T

	for attempt := 0; attempt < maxAggregateWriteRetries; attempt++ {
		aggregate, exists, err := loadAggregateForUpdate(ctx, store, packageName, allowCreate)
		if err != nil {
			return zero, err
		}

		updated, result, err := mutate(aggregate, exists)
		if err != nil {
			return zero, err
		}
		if !aggregateMutationChanged(aggregate, updated) {
			return result, nil
		}

		if err := store.PutAggregate(ctx, updated, aggregate.Revision); err != nil {
			if errors.Is(err, ErrAggregateRevisionConflict) {
				continue
			}
			return zero, err
		}
		return result, nil
	}

	return zero, NewError(409, "concurrent_modification", "Concurrent catalog update conflict.", map[string]any{"packageName": packageName})
}

func loadAggregateForUpdate(ctx context.Context, store AggregateStateStore, packageName string, allowCreate bool) (AgentSkillAggregate, bool, error) {
	aggregate, err := store.GetAggregate(ctx, packageName)
	if err == nil {
		return aggregate, true, nil
	}
	if errors.Is(err, ErrAgentSkillAggregateNotFound) {
		if allowCreate {
			return AgentSkillAggregate{}, false, nil
		}
		return AgentSkillAggregate{}, false, NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}
	return AgentSkillAggregate{}, false, err
}

func aggregateMutationChanged(before AgentSkillAggregate, after AgentSkillAggregate) bool {
	return !reflect.DeepEqual(before.Skill, after.Skill) || !reflect.DeepEqual(before.Governance, after.Governance)
}
