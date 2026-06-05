package catalog

import (
	"context"
)

type ReadModule struct {
	store QueryStore
}

func newReadModule(store QueryStore) *ReadModule {
	return &ReadModule{store: store}
}

func (m *ReadModule) ListSkills(ctx context.Context, query SkillQuery) (SkillsPage, error) {
	return m.store.ListSkills(ctx, query)
}

func (m *ReadModule) ListCuratedSkills(ctx context.Context, pageSize int, cursor int) (CuratedSkillsPage, error) {
	return m.store.ListCuratedSkills(ctx, pageSize, cursor)
}

func (m *ReadModule) GetSkill(ctx context.Context, packageName string) (SkillDetail, error) {
	return m.store.GetSkill(ctx, packageName)
}

func (m *ReadModule) ListVersions(ctx context.Context, query VersionQuery) (VersionsPage, error) {
	return m.store.ListVersions(ctx, query)
}

func (m *ReadModule) GetVersion(ctx context.Context, packageName string, version string) (SkillVersionDetail, error) {
	return m.store.GetVersion(ctx, packageName, version)
}

func (m *ReadModule) ListGovernanceChanges(ctx context.Context, query GovernanceQuery) (GovernancePage, error) {
	if query.TargetType != "" && query.TargetType != string(GovernanceTargetSkill) && query.TargetType != string(GovernanceTargetVersion) {
		return GovernancePage{}, NewError(400, "invalid_request", "target_type is invalid.", map[string]any{"targetType": query.TargetType})
	}
	return m.store.ListGovernanceChanges(ctx, query)
}

func (m *ReadModule) ResolveInstallTarget(ctx context.Context, packageName string, version string) (InstallResolution, error) {
	skill, err := m.store.GetSkill(ctx, packageName)
	if err != nil {
		return InstallResolution{}, err
	}
	if skill.Status == SkillStatusRetired {
		return InstallResolution{}, NewError(409, "skill_retired", "Skill is retired.", map[string]any{"packageName": packageName})
	}

	versions, err := m.store.ListPackageVersions(ctx, packageName)
	if err != nil {
		return InstallResolution{}, err
	}

	target, err := m.resolveInstallVersion(packageName, version, versions)
	if err != nil {
		return InstallResolution{}, err
	}

	if target.VersionStatus == VersionStatusYanked {
		return InstallResolution{}, NewError(409, "version_yanked", "Version is yanked.", m.versionErrorDetails(ctx, skill, target.Version))
	}
	if target.InstallPolicy != InstallPolicyResolvable {
		return InstallResolution{}, NewError(409, "not_resolvable_by_policy", "Version is not resolvable by current install policy.", m.versionErrorDetails(ctx, skill, target.Version))
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

func (m *ReadModule) resolveInstallVersion(packageName string, version string, versions []SkillVersionDetail) (SkillVersionDetail, error) {
	if version != "" {
		if !isExactVersion(version) {
			return SkillVersionDetail{}, NewError(422, "invalid_request", "version must be an exact semver string.", map[string]any{"version": version})
		}
		for _, item := range versions {
			if item.Version == version {
				return item, nil
			}
		}
		return SkillVersionDetail{}, NewError(404, "version_not_found", "Version not found.", map[string]any{
			"packageName": packageName,
			"version":     version,
		})
	}

	var target *SkillVersionDetail
	for _, item := range versions {
		if item.VersionStatus != VersionStatusPublished || item.InstallPolicy != InstallPolicyResolvable {
			continue
		}
		if target == nil || compareExactVersions(item.Version, target.Version) > 0 {
			candidate := item
			target = &candidate
		}
	}
	if target == nil {
		return SkillVersionDetail{}, NewError(409, "not_resolvable_by_policy", "No installable version is currently resolvable.", map[string]any{"packageName": packageName})
	}
	return *target, nil
}

func (m *ReadModule) versionErrorDetails(ctx context.Context, skill SkillDetail, version string) map[string]any {
	details := map[string]any{
		"packageName": skill.PackageName,
		"version":     version,
	}

	governance, err := m.store.ListPackageGovernance(ctx, skill.PackageName, GovernanceTargetVersion, stringPtr(version))
	if err != nil {
		return details
	}

	if recommendation := latestReplacementRecommendation(skill.PackageName, version, governance); recommendation != nil {
		details["replacementRecommendation"] = *recommendation
	}
	return details
}
