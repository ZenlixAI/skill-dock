package application

import (
	"context"
	"errors"
	"strings"

	"skill_registry/internal/catalog/domain"
)

type ReadService struct {
	store domain.QueryStore
}

func NewReadService(store domain.QueryStore) *ReadService {
	return &ReadService{store: store}
}

func (s *ReadService) ListSkills(ctx context.Context, query domain.SkillQuery) (domain.SkillsPage, error) {
	return s.store.ListSkills(ctx, query)
}

func (s *ReadService) ListCuratedSkills(ctx context.Context, query domain.CuratedSkillQuery) (domain.CuratedSkillsPage, error) {
	return s.store.ListCuratedSkills(ctx, query)
}

func (s *ReadService) GetSkill(ctx context.Context, packageName string) (domain.SkillDetail, error) {
	return s.store.GetSkill(ctx, packageName)
}

func (s *ReadService) ListVersions(ctx context.Context, query domain.VersionQuery) (domain.VersionsPage, error) {
	return s.store.ListVersions(ctx, query)
}

func (s *ReadService) GetVersion(ctx context.Context, packageName string, version string) (domain.SkillVersionDetail, error) {
	return s.store.GetVersion(ctx, packageName, version)
}

func (s *ReadService) ListGovernanceChanges(ctx context.Context, query domain.GovernanceQuery) (domain.GovernancePage, error) {
	if query.TargetType != "" && query.TargetType != string(domain.GovernanceTargetSkill) && query.TargetType != string(domain.GovernanceTargetVersion) {
		return domain.GovernancePage{}, domain.NewError(400, "invalid_request", "target_type is invalid.", map[string]any{"targetType": query.TargetType})
	}
	return s.store.ListGovernanceChanges(ctx, query)
}

func (s *ReadService) ResolveInstallTarget(ctx context.Context, packageName string, version string) (domain.InstallResolution, error) {
	skill, err := s.store.GetSkill(ctx, packageName)
	if err != nil {
		return domain.InstallResolution{}, err
	}
	if skill.Status == domain.SkillStatusRetired {
		return domain.InstallResolution{}, domain.NewError(409, "skill_retired", "Skill is retired.", map[string]any{"packageName": packageName})
	}

	target, err := s.resolveInstallVersion(ctx, packageName, version)
	if err != nil {
		return domain.InstallResolution{}, err
	}

	if target.VersionStatus == domain.VersionStatusYanked {
		return domain.InstallResolution{}, domain.NewError(409, "version_yanked", "Version is yanked.", s.versionErrorDetails(ctx, skill, target.Version))
	}
	if target.InstallPolicy != domain.InstallPolicyResolvable {
		return domain.InstallResolution{}, domain.NewError(409, "not_resolvable_by_policy", "Version is not resolvable by current install policy.", s.versionErrorDetails(ctx, skill, target.Version))
	}

	return domain.InstallResolution{
		PackageName:         target.PackageName,
		Version:             target.Version,
		NPMRegistry:         target.Registry.NPMRegistry,
		Integrity:           target.Registry.Integrity,
		TarballURL:          target.Registry.TarballURL,
		SkillManifestDigest: target.Bundle.SkillManifestDigest,
		VersionStatus:       target.VersionStatus,
	}, nil
}

func (s *ReadService) resolveInstallVersion(ctx context.Context, packageName string, version string) (domain.SkillVersionDetail, error) {
	if version != "" {
		if !isExactVersion(version) {
			return domain.SkillVersionDetail{}, domain.NewError(422, "invalid_request", "version must be an exact semver string.", map[string]any{"version": version})
		}
		return s.store.GetVersion(ctx, packageName, version)
	}

	target, err := s.store.GetLatestResolvableVersion(ctx, packageName)
	if err == nil {
		return target, nil
	}
	if errors.Is(err, domain.ErrNoResolvableVersion) {
		return domain.SkillVersionDetail{}, domain.NewError(409, "not_resolvable_by_policy", "No installable version is currently resolvable.", map[string]any{"packageName": packageName})
	}
	return domain.SkillVersionDetail{}, err
}

func (s *ReadService) versionErrorDetails(ctx context.Context, skill domain.SkillDetail, version string) map[string]any {
	details := map[string]any{
		"packageName": skill.PackageName,
		"version":     version,
	}

	governance, err := s.store.ListPackageGovernance(ctx, skill.PackageName, domain.GovernanceTargetVersion, stringPtr(version))
	if err != nil {
		return details
	}

	if recommendation := latestReplacementRecommendation(skill.PackageName, version, governance); recommendation != nil {
		details["replacementRecommendation"] = *recommendation
	}
	return details
}

func latestReplacementRecommendation(packageName string, version string, records []domain.GovernanceRecord) *string {
	for i := len(records) - 1; i >= 0; i-- {
		record := records[i]
		if record.PackageName != packageName {
			continue
		}
		if record.Version == nil || *record.Version != version {
			continue
		}
		if record.ReplacementRecommendation != nil {
			return record.ReplacementRecommendation
		}
	}
	return nil
}

func isExactVersion(version string) bool {
	parts := strings.Split(strings.TrimSpace(version), ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}
	return true
}

func stringPtr(value string) *string {
	return &value
}
