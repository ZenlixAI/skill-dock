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

type CatalogRepository struct {
	stateStore CatalogStateStore
	artifacts  ArtifactSource
}

type CatalogRepositoryConfig struct {
	StateStore     CatalogStateStore
	ArtifactSource ArtifactSource
}

func NewCatalogRepository(config CatalogRepositoryConfig) *CatalogRepository {
	if config.StateStore == nil {
		panic("catalog state store is required")
	}
	return &CatalogRepository{
		stateStore: config.StateStore,
		artifacts:  config.ArtifactSource,
	}
}

func (r *CatalogRepository) Ping(ctx context.Context) error {
	return r.stateStore.Ping(ctx)
}

func (r *CatalogRepository) ListSkills(ctx context.Context, query SkillQuery) (SkillsPage, error) {
	return r.stateStore.ListSkills(ctx, query)
}

func (r *CatalogRepository) ListCuratedSkills(ctx context.Context, pageSize int, cursor int) (CuratedSkillsPage, error) {
	return r.stateStore.ListCuratedSkills(ctx, pageSize, cursor)
}

func (r *CatalogRepository) GetSkill(ctx context.Context, packageName string) (SkillDetail, error) {
	aggregate, err := r.getAggregate(ctx, packageName)
	if err != nil {
		return SkillDetail{}, err
	}
	return aggregate.Skill.Detail, nil
}

func (r *CatalogRepository) ListVersions(ctx context.Context, query VersionQuery) (VersionsPage, error) {
	return r.stateStore.ListVersions(ctx, query)
}

func (r *CatalogRepository) GetVersion(ctx context.Context, packageName string, version string) (SkillVersionDetail, error) {
	aggregate, err := r.getAggregate(ctx, packageName)
	if err != nil {
		return SkillVersionDetail{}, err
	}

	for _, item := range aggregate.Skill.Versions {
		if item.Version == version {
			return item, nil
		}
	}

	return SkillVersionDetail{}, NewError(404, "version_not_found", "Version not found.", map[string]any{
		"packageName": packageName,
		"version":     version,
	})
}

func (r *CatalogRepository) ListGovernanceChanges(ctx context.Context, query GovernanceQuery) (GovernancePage, error) {
	if query.TargetType != "" && query.TargetType != string(GovernanceTargetSkill) && query.TargetType != string(GovernanceTargetVersion) {
		return GovernancePage{}, NewError(400, "invalid_request", "target_type is invalid.", map[string]any{"targetType": query.TargetType})
	}
	return r.stateStore.ListGovernanceChanges(ctx, query)
}

func (r *CatalogRepository) ResolveInstallTarget(ctx context.Context, packageName string, version string) (InstallResolution, error) {
	aggregate, err := r.getAggregate(ctx, packageName)
	if err != nil {
		return InstallResolution{}, err
	}
	record := aggregateSkillRecord(aggregate)

	if record.Detail.Status == SkillStatusRetired {
		return InstallResolution{}, NewError(409, "skill_retired", "Skill is retired.", map[string]any{"packageName": packageName})
	}

	var target *SkillVersionDetail
	if version != "" {
		if !isExactVersion(version) {
			return InstallResolution{}, NewError(422, "invalid_request", "version must be an exact semver string.", map[string]any{"version": version})
		}
		for _, item := range record.Versions {
			if item.Version == version {
				candidate := item
				target = &candidate
				break
			}
		}
		if target == nil {
			return InstallResolution{}, NewError(404, "version_not_found", "Version not found.", map[string]any{
				"packageName": packageName,
				"version":     version,
			})
		}
	} else {
		for _, item := range record.Versions {
			if item.VersionStatus != VersionStatusPublished || item.InstallPolicy != InstallPolicyResolvable {
				continue
			}
			if target == nil || compareExactVersions(item.Version, target.Version) > 0 {
				candidate := item
				target = &candidate
			}
		}
		if target == nil {
			return InstallResolution{}, NewError(409, "not_resolvable_by_policy", "No installable version is currently resolvable.", map[string]any{"packageName": packageName})
		}
	}

	if target.VersionStatus == VersionStatusYanked {
		return InstallResolution{}, NewError(409, "version_yanked", "Version is yanked.", versionErrorDetails(record, target.Version, aggregate.Governance))
	}
	if target.InstallPolicy != InstallPolicyResolvable {
		return InstallResolution{}, NewError(409, "not_resolvable_by_policy", "Version is not resolvable by current install policy.", versionErrorDetails(record, target.Version, aggregate.Governance))
	}

	return InstallResolution{
		PackageName:         target.PackageName,
		Version:             target.Version,
		NPMRegistry:         target.Registry.NPMRegistry,
		Integrity:           target.Registry.Integrity,
		TarballURL:          target.Registry.TarballURL,
		SkillManifestDigest: target.Bundle.SkillManifestDigest,
		VersionStatus:       target.VersionStatus,
	}, nil
}

func (r *CatalogRepository) CreateReleaseRecord(ctx context.Context, packageName string, version string, input ReleaseRecordInput) (ReleaseRecordResult, error) {
	if !isExactVersion(version) {
		return ReleaseRecordResult{}, NewError(422, "invalid_request", "version must be an exact semver string.", map[string]any{"version": version})
	}
	if r.artifacts == nil {
		return ReleaseRecordResult{}, NewError(500, "internal_error", "Artifact source is not configured.", nil)
	}

	artifact, err := r.artifacts.FindReleaseArtifact(ctx, packageName, version)
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

	return updateAggregate(ctx, r.stateStore, packageName, true, func(aggregate AgentSkillAggregate, exists bool) (AgentSkillAggregate, ReleaseRecordResult, error) {
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

func (r *CatalogRepository) ApplySkillGovernanceChange(ctx context.Context, packageName string, input SkillGovernanceChangeInput, actor string) (GovernanceChangeResponse, error) {
	if input.Action != GovernanceActionRetire {
		return GovernanceChangeResponse{}, NewError(422, "invalid_request", "skill governance action is invalid.", map[string]any{"action": input.Action})
	}
	if strings.TrimSpace(input.Reason) == "" {
		return GovernanceChangeResponse{}, NewError(422, "invalid_request", "reason is required.", map[string]any{"field": "reason"})
	}

	return updateAggregate(ctx, r.stateStore, packageName, false, func(aggregate AgentSkillAggregate, _ bool) (AgentSkillAggregate, GovernanceChangeResponse, error) {
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

func (r *CatalogRepository) ApplyVersionGovernanceChange(ctx context.Context, packageName string, version string, input VersionGovernanceChangeInput, actor string) (GovernanceChangeResponse, error) {
	if !isExactVersion(version) {
		return GovernanceChangeResponse{}, NewError(422, "invalid_request", "version must be an exact semver string.", map[string]any{"version": version})
	}
	if input.Action != GovernanceActionDeprecate && input.Action != GovernanceActionYank {
		return GovernanceChangeResponse{}, NewError(422, "invalid_request", "version governance action is invalid.", map[string]any{"action": input.Action})
	}
	if strings.TrimSpace(input.Reason) == "" {
		return GovernanceChangeResponse{}, NewError(422, "invalid_request", "reason is required.", map[string]any{"field": "reason"})
	}

	return updateAggregate(ctx, r.stateStore, packageName, false, func(aggregate AgentSkillAggregate, _ bool) (AgentSkillAggregate, GovernanceChangeResponse, error) {
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

func (r *CatalogRepository) Seed(ctx context.Context, data SampleCatalogData) error {
	if len(data.Skills) == 0 {
		return nil
	}
	empty, err := r.stateStore.IsEmpty(ctx)
	if err != nil {
		return err
	}
	if !empty {
		return nil
	}
	for _, aggregate := range aggregatesFromSampleData(data) {
		if err := r.stateStore.PutAggregate(ctx, aggregate, 0); err != nil {
			if errors.Is(err, ErrAggregateRevisionConflict) {
				return nil
			}
			return err
		}
	}
	return nil
}

func (r *CatalogRepository) getAggregate(ctx context.Context, packageName string) (AgentSkillAggregate, error) {
	aggregate, err := r.stateStore.GetAggregate(ctx, packageName)
	if err != nil {
		if errors.Is(err, ErrAgentSkillAggregateNotFound) {
			return AgentSkillAggregate{}, NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
		}
		return AgentSkillAggregate{}, err
	}
	return aggregate, nil
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

func versionErrorDetails(record skillRecord, version string, governance []GovernanceRecord) map[string]any {
	details := map[string]any{
		"packageName": record.Detail.PackageName,
		"version":     version,
	}
	if recommendation := latestReplacementRecommendation(record.Detail.PackageName, version, governance); recommendation != nil {
		details["replacementRecommendation"] = *recommendation
	}
	return details
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
	store CatalogStateStore,
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

func loadAggregateForUpdate(ctx context.Context, store CatalogStateStore, packageName string, allowCreate bool) (AgentSkillAggregate, bool, error) {
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
