package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"skill_registry/internal/catalog/domain"
)

const maxAggregateWriteRetries = 5

type WriteService struct {
	stateStore domain.AggregateStateStore
	artifacts  domain.ArtifactSource
}

func NewWriteService(stateStore domain.AggregateStateStore, artifacts domain.ArtifactSource) *WriteService {
	return &WriteService{
		stateStore: stateStore,
		artifacts:  artifacts,
	}
}

func (s *WriteService) Ping(ctx context.Context) error {
	return s.stateStore.Ping(ctx)
}

func (s *WriteService) Seed(ctx context.Context, data domain.SampleCatalogData) error {
	if len(data.Skills) == 0 {
		return nil
	}
	empty, err := s.stateStore.IsEmpty(ctx)
	if err != nil {
		return err
	}
	if !empty {
		return nil
	}
	for _, aggregate := range aggregatesFromSampleData(data) {
		if err := s.stateStore.PutAggregate(ctx, aggregate, 0); err != nil {
			if errors.Is(err, domain.ErrAggregateRevisionConflict) {
				return nil
			}
			return err
		}
	}
	return nil
}

func (s *WriteService) CreateReleaseRecord(ctx context.Context, packageName string, version string, input domain.ReleaseRecordInput) (domain.ReleaseRecordResult, error) {
	if !isExactVersion(version) {
		return domain.ReleaseRecordResult{}, domain.NewError(422, "invalid_request", "version must be an exact semver string.", map[string]any{"version": version})
	}
	if s.artifacts == nil {
		return domain.ReleaseRecordResult{}, domain.NewError(500, "internal_error", "Artifact source is not configured.", nil)
	}

	artifact, err := s.artifacts.FindReleaseArtifact(ctx, packageName, version)
	if err != nil {
		return domain.ReleaseRecordResult{}, err
	}
	if artifact.RegistryDown {
		return domain.ReleaseRecordResult{}, domain.NewError(503, "registry_unavailable", "Registry is unavailable.", map[string]any{
			"packageName": packageName,
			"version":     version,
		})
	}
	if artifact.ValidationError != "" {
		return domain.ReleaseRecordResult{}, domain.NewError(422, "bundle_validation_failed", "Bundle validation failed.", map[string]any{
			"packageName": packageName,
			"version":     version,
			"reason":      artifact.ValidationError,
		})
	}
	if validationError := validateReleaseRecordInput(input, artifact); validationError != nil {
		return domain.ReleaseRecordResult{}, validationError
	}

	return updateAggregate(ctx, s.stateStore, packageName, true, func(aggregate domain.AgentSkillAggregate, exists bool) (domain.AgentSkillAggregate, domain.ReleaseRecordResult, error) {
		record := aggregateSkillRecord(aggregate)
		if !exists {
			record = domain.SkillRecord{
				Detail: domain.SkillDetail{
					SkillSummary: domain.SkillSummary{
						PackageName: packageName,
						DisplayName: artifact.SkillDisplayName,
						Description: artifact.SkillDescription,
						SourceType:  artifact.SkillSourceType,
						Status:      domain.SkillStatusActive,
					},
					CreatedAt: artifact.Version.Registry.PublishedAt,
				},
			}
		}

		for _, existing := range record.Versions {
			if existing.Version != version {
				continue
			}
			if sameReleaseContent(existing, artifact.Version) {
				return aggregate, domain.ReleaseRecordResult{
					PackageName:      packageName,
					Version:          version,
					Created:          false,
					IdempotentReplay: true,
					VersionStatus:    existing.VersionStatus,
					InstallPolicy:    existing.InstallPolicy,
				}, nil
			}
			return domain.AgentSkillAggregate{}, domain.ReleaseRecordResult{}, domain.NewError(409, "immutable_release_conflict", "Immutable release content conflicts with the existing record.", map[string]any{
				"packageName": packageName,
				"version":     version,
			})
		}

		record.Versions = append(record.Versions, artifact.Version)
		record.Detail.UpdatedAt = artifact.Version.Registry.PublishedAt
		refreshLatestVersion(&record)

		updated := aggregateFromSkillRecord(record)
		updated.Revision = aggregate.Revision
		updated.Governance = append([]domain.GovernanceRecord(nil), aggregate.Governance...)
		return updated, domain.ReleaseRecordResult{
			PackageName:      packageName,
			Version:          version,
			Created:          true,
			IdempotentReplay: false,
			VersionStatus:    artifact.Version.VersionStatus,
			InstallPolicy:    artifact.Version.InstallPolicy,
		}, nil
	})
}

func sameReleaseContent(existing domain.SkillVersionDetail, candidate domain.SkillVersionDetail) bool {
	existing.PublishedAt = time.Time{}
	existing.Registry.PublishedAt = time.Time{}
	candidate.PublishedAt = time.Time{}
	candidate.Registry.PublishedAt = time.Time{}
	return reflect.DeepEqual(existing, candidate)
}

func (s *WriteService) ApplySkillGovernanceChange(ctx context.Context, packageName string, input domain.SkillGovernanceChangeInput, actor string) (domain.GovernanceChangeResponse, error) {
	if input.Action != domain.GovernanceActionRetire {
		return domain.GovernanceChangeResponse{}, domain.NewError(422, "invalid_request", "skill governance action is invalid.", map[string]any{"action": input.Action})
	}
	if strings.TrimSpace(input.Reason) == "" {
		return domain.GovernanceChangeResponse{}, domain.NewError(422, "invalid_request", "reason is required.", map[string]any{"field": "reason"})
	}

	return updateAggregate(ctx, s.stateStore, packageName, false, func(aggregate domain.AgentSkillAggregate, _ bool) (domain.AgentSkillAggregate, domain.GovernanceChangeResponse, error) {
		record := aggregateSkillRecord(aggregate)
		if record.Detail.Status == domain.SkillStatusRetired {
			return domain.AgentSkillAggregate{}, domain.GovernanceChangeResponse{}, domain.NewError(409, "invalid_state_transition", "Skill is already retired.", map[string]any{"packageName": packageName})
		}

		record.Detail.Status = domain.SkillStatusRetired
		record.Detail.UpdatedAt = time.Now().UTC()

		governanceRecord := domain.GovernanceRecord{
			ID:          nextGovernanceID(packageName, aggregate.Revision+1),
			TargetType:  domain.GovernanceTargetSkill,
			PackageName: packageName,
			Action:      domain.GovernanceActionRetire,
			Reason:      input.Reason,
			ChangedBy:   defaultActor(actor),
			ChangedAt:   record.Detail.UpdatedAt,
		}

		updated := aggregateFromSkillRecord(record)
		updated.Revision = aggregate.Revision
		updated.Governance = append(append([]domain.GovernanceRecord(nil), aggregate.Governance...), governanceRecord)

		status := domain.SkillStatusRetired
		return updated, domain.GovernanceChangeResponse{
			Record: governanceRecord,
			ResultingState: domain.GovernanceChangeResultingState{
				Status: &status,
			},
		}, nil
	})
}

func (s *WriteService) ApplyVersionGovernanceChange(ctx context.Context, packageName string, version string, input domain.VersionGovernanceChangeInput, actor string) (domain.GovernanceChangeResponse, error) {
	if !isExactVersion(version) {
		return domain.GovernanceChangeResponse{}, domain.NewError(422, "invalid_request", "version must be an exact semver string.", map[string]any{"version": version})
	}
	if input.Action != domain.GovernanceActionDeprecate && input.Action != domain.GovernanceActionYank {
		return domain.GovernanceChangeResponse{}, domain.NewError(422, "invalid_request", "version governance action is invalid.", map[string]any{"action": input.Action})
	}
	if strings.TrimSpace(input.Reason) == "" {
		return domain.GovernanceChangeResponse{}, domain.NewError(422, "invalid_request", "reason is required.", map[string]any{"field": "reason"})
	}

	return updateAggregate(ctx, s.stateStore, packageName, false, func(aggregate domain.AgentSkillAggregate, _ bool) (domain.AgentSkillAggregate, domain.GovernanceChangeResponse, error) {
		record := aggregateSkillRecord(aggregate)

		index := -1
		for i, item := range record.Versions {
			if item.Version == version {
				index = i
				break
			}
		}
		if index < 0 {
			return domain.AgentSkillAggregate{}, domain.GovernanceChangeResponse{}, domain.NewError(404, "version_not_found", "Version not found.", map[string]any{
				"packageName": packageName,
				"version":     version,
			})
		}

		current := record.Versions[index]
		switch input.Action {
		case domain.GovernanceActionDeprecate:
			if current.VersionStatus != domain.VersionStatusPublished {
				return domain.AgentSkillAggregate{}, domain.GovernanceChangeResponse{}, domain.NewError(409, "invalid_state_transition", "Only published versions can be deprecated.", map[string]any{
					"packageName": packageName,
					"version":     version,
				})
			}
			current.VersionStatus = domain.VersionStatusDeprecated
		case domain.GovernanceActionYank:
			if current.VersionStatus == domain.VersionStatusYanked {
				return domain.AgentSkillAggregate{}, domain.GovernanceChangeResponse{}, domain.NewError(409, "invalid_state_transition", "Version is already yanked.", map[string]any{
					"packageName": packageName,
					"version":     version,
				})
			}
			current.VersionStatus = domain.VersionStatusYanked
			current.InstallPolicy = domain.InstallPolicyBlocked
		}

		record.Versions[index] = current
		record.Detail.UpdatedAt = time.Now().UTC()
		refreshLatestVersion(&record)

		governanceRecord := domain.GovernanceRecord{
			ID:                        nextGovernanceID(packageName, aggregate.Revision+1),
			TargetType:                domain.GovernanceTargetVersion,
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
		updated.Governance = append(append([]domain.GovernanceRecord(nil), aggregate.Governance...), governanceRecord)

		resultingStatus := current.VersionStatus
		return updated, domain.GovernanceChangeResponse{
			Record: governanceRecord,
			ResultingState: domain.GovernanceChangeResultingState{
				VersionStatus: &resultingStatus,
			},
		}, nil
	})
}

func aggregateSkillRecord(aggregate domain.AgentSkillAggregate) domain.SkillRecord {
	return domain.SkillRecord{
		Detail:         aggregate.Skill.Detail,
		CurationReason: aggregate.Skill.CurationReason,
		Versions:       append([]domain.SkillVersionDetail(nil), aggregate.Skill.Versions...),
	}
}

func aggregateFromSkillRecord(record domain.SkillRecord) domain.AgentSkillAggregate {
	return domain.AgentSkillAggregate{
		Skill: domain.SkillSnapshot{
			Detail:         record.Detail,
			CurationReason: record.CurationReason,
			Versions:       append([]domain.SkillVersionDetail(nil), record.Versions...),
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
	store domain.AggregateStateStore,
	packageName string,
	allowCreate bool,
	mutate func(aggregate domain.AgentSkillAggregate, exists bool) (domain.AgentSkillAggregate, T, error),
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
			if errors.Is(err, domain.ErrAggregateRevisionConflict) {
				continue
			}
			return zero, err
		}
		return result, nil
	}

	return zero, domain.NewError(409, "concurrent_modification", "Concurrent catalog update conflict.", map[string]any{"packageName": packageName})
}

func loadAggregateForUpdate(ctx context.Context, store domain.AggregateStateStore, packageName string, allowCreate bool) (domain.AgentSkillAggregate, bool, error) {
	aggregate, err := store.GetAggregate(ctx, packageName)
	if err == nil {
		return aggregate, true, nil
	}
	if errors.Is(err, domain.ErrAgentSkillAggregateNotFound) {
		if allowCreate {
			return domain.AgentSkillAggregate{}, false, nil
		}
		return domain.AgentSkillAggregate{}, false, domain.NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}
	return domain.AgentSkillAggregate{}, false, err
}

func aggregateMutationChanged(before domain.AgentSkillAggregate, after domain.AgentSkillAggregate) bool {
	return !reflect.DeepEqual(before.Skill, after.Skill) || !reflect.DeepEqual(before.Governance, after.Governance)
}

func aggregatesFromSampleData(data domain.SampleCatalogData) []domain.AgentSkillAggregate {
	aggregates := make([]domain.AgentSkillAggregate, 0, len(data.Skills))
	governanceByPackage := make(map[string][]domain.GovernanceRecord)
	for _, record := range data.Governance {
		governanceByPackage[record.PackageName] = append(governanceByPackage[record.PackageName], record)
	}
	for _, skill := range data.Skills {
		aggregates = append(aggregates, domain.AgentSkillAggregate{
			Revision: 1,
			Skill: domain.SkillSnapshot{
				Detail:         skill.Detail,
				CurationReason: skill.CurationReason,
				Versions:       append([]domain.SkillVersionDetail(nil), skill.Versions...),
			},
			Governance: append([]domain.GovernanceRecord(nil), governanceByPackage[skill.Detail.PackageName]...),
		})
	}
	sort.Slice(aggregates, func(i int, j int) bool {
		return aggregates[i].Skill.Detail.PackageName < aggregates[j].Skill.Detail.PackageName
	})
	return aggregates
}

func refreshLatestVersion(record *domain.SkillRecord) {
	var latest *domain.SkillVersionDetail
	for _, version := range record.Versions {
		if version.VersionStatus == domain.VersionStatusYanked {
			continue
		}
		if latest == nil || compareExactVersions(version.Version, latest.Version) > 0 {
			candidate := version
			latest = &candidate
		}
	}
	if latest == nil {
		for _, version := range record.Versions {
			if latest == nil || compareExactVersions(version.Version, latest.Version) > 0 {
				candidate := version
				latest = &candidate
			}
		}
	}
	if latest == nil {
		record.Detail.LatestVersion = nil
		record.Detail.LatestVersionStatus = nil
		return
	}

	record.Detail.LatestVersion = stringPtr(latest.Version)
	record.Detail.LatestVersionStatus = stringPtr(string(latest.VersionStatus))
}

func compareExactVersions(left string, right string) int {
	lMajor, lMinor, lPatch := parseVersionCore(left)
	rMajor, rMinor, rPatch := parseVersionCore(right)

	if lMajor != rMajor {
		return compareInts(lMajor, rMajor)
	}
	if lMinor != rMinor {
		return compareInts(lMinor, rMinor)
	}
	return compareInts(lPatch, rPatch)
}

func parseVersionCore(version string) (int, int, int) {
	core := version
	if idx := strings.Index(core, "-"); idx >= 0 {
		core = core[:idx]
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return 0, 0, 0
	}

	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])
	return major, minor, patch
}

func compareInts(left int, right int) int {
	switch {
	case left > right:
		return 1
	case left < right:
		return -1
	default:
		return 0
	}
}

func defaultActor(actor string) string {
	if strings.TrimSpace(actor) == "" {
		return "system"
	}
	return actor
}

func validateReleaseRecordInput(input domain.ReleaseRecordInput, artifact domain.ReleaseArtifact) *domain.Error {
	if input.PublishRegistry != nil && strings.TrimSpace(*input.PublishRegistry) != artifact.Version.Registry.NPMRegistry {
		return domain.NewError(422, "invalid_request", "publishRegistry does not match the resolved registry metadata.", map[string]any{
			"publishRegistry": *input.PublishRegistry,
			"npmRegistry":     artifact.Version.Registry.NPMRegistry,
		})
	}
	if input.ManifestDigest != nil && strings.TrimSpace(*input.ManifestDigest) != artifact.Version.Bundle.SkillManifestDigest {
		return domain.NewError(422, "invalid_request", "manifestDigest does not match the resolved bundle metadata.", map[string]any{
			"manifestDigest":      *input.ManifestDigest,
			"skillManifestDigest": artifact.Version.Bundle.SkillManifestDigest,
		})
	}
	if len(input.Components) > 0 && !sameStringSet(input.Components, artifact.Version.Bundle.Components) {
		return domain.NewError(422, "invalid_request", "components do not match the resolved bundle metadata.", map[string]any{
			"components":         input.Components,
			"resolvedComponents": artifact.Version.Bundle.Components,
		})
	}
	return nil
}

func sameStringSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	normalize := func(values []string) []string {
		result := append([]string(nil), values...)
		for i := range result {
			result[i] = strings.TrimSpace(result[i])
		}
		sort.Strings(result)
		return result
	}
	leftNormalized := normalize(left)
	rightNormalized := normalize(right)
	for i := range leftNormalized {
		if leftNormalized[i] != rightNormalized[i] {
			return false
		}
	}
	return true
}
