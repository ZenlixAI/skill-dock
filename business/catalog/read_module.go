package catalog

import (
	"context"
	"errors"
)

var errNoResolvableVersion = errors.New("no resolvable version")

type ReadModule struct {
	store QueryStore
}

func newReadModule(store QueryStore) *ReadModule {
	return &ReadModule{store: store}
}

func (m *ReadModule) ListSkills(ctx context.Context, query SkillQuery) (SkillsPage, error) {
	return m.store.ListSkills(ctx, query)
}

func (m *ReadModule) ListCuratedSkills(ctx context.Context, query CuratedSkillQuery) (CuratedSkillsPage, error) {
	return m.store.ListCuratedSkills(ctx, query)
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

	target, err := m.resolveInstallVersion(ctx, packageName, version)
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

func (m *ReadModule) resolveInstallVersion(ctx context.Context, packageName string, version string) (SkillVersionDetail, error) {
	if version != "" {
		if !isExactVersion(version) {
			return SkillVersionDetail{}, NewError(422, "invalid_request", "version must be an exact semver string.", map[string]any{"version": version})
		}
		return m.store.GetVersion(ctx, packageName, version)
	}

	target, err := m.store.GetLatestResolvableVersion(ctx, packageName)
	if err == nil {
		return target, nil
	}
	if errors.Is(err, errNoResolvableVersion) {
		return SkillVersionDetail{}, NewError(409, "not_resolvable_by_policy", "No installable version is currently resolvable.", map[string]any{"packageName": packageName})
	}
	return SkillVersionDetail{}, err
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
