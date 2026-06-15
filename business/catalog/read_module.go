package catalog

import (
	"context"
	"errors"

	internalapp "skill_registry/internal/catalog/application"
	internaldomain "skill_registry/internal/catalog/domain"
)

var errNoResolvableVersion = errors.New("no resolvable version")

type ReadModule struct {
	store   QueryStore
	service *internalapp.ReadService
}

func newReadModule(store QueryStore) *ReadModule {
	return &ReadModule{
		store:   store,
		service: internalapp.NewReadService(&queryStoreAdapter{store: store}),
	}
}

func (m *ReadModule) ListSkills(ctx context.Context, query SkillQuery) (SkillsPage, error) {
	return m.service.ListSkills(ctx, query)
}

func (m *ReadModule) ListCuratedSkills(ctx context.Context, query CuratedSkillQuery) (CuratedSkillsPage, error) {
	return m.service.ListCuratedSkills(ctx, query)
}

func (m *ReadModule) GetSkill(ctx context.Context, packageName string) (SkillDetail, error) {
	return m.service.GetSkill(ctx, packageName)
}

func (m *ReadModule) ListVersions(ctx context.Context, query VersionQuery) (VersionsPage, error) {
	return m.service.ListVersions(ctx, query)
}

func (m *ReadModule) GetVersion(ctx context.Context, packageName string, version string) (SkillVersionDetail, error) {
	return m.service.GetVersion(ctx, packageName, version)
}

func (m *ReadModule) ListGovernanceChanges(ctx context.Context, query GovernanceQuery) (GovernancePage, error) {
	return m.service.ListGovernanceChanges(ctx, query)
}

func (m *ReadModule) ResolveInstallTarget(ctx context.Context, packageName string, version string) (InstallResolution, error) {
	return m.service.ResolveInstallTarget(ctx, packageName, version)
}

type queryStoreAdapter struct {
	store QueryStore
}

func (a *queryStoreAdapter) ListSkills(ctx context.Context, query internaldomain.SkillQuery) (internaldomain.SkillsPage, error) {
	return a.store.ListSkills(ctx, query)
}

func (a *queryStoreAdapter) ListCuratedSkills(ctx context.Context, query internaldomain.CuratedSkillQuery) (internaldomain.CuratedSkillsPage, error) {
	return a.store.ListCuratedSkills(ctx, query)
}

func (a *queryStoreAdapter) GetSkill(ctx context.Context, packageName string) (internaldomain.SkillDetail, error) {
	return a.store.GetSkill(ctx, packageName)
}

func (a *queryStoreAdapter) ListVersions(ctx context.Context, query internaldomain.VersionQuery) (internaldomain.VersionsPage, error) {
	return a.store.ListVersions(ctx, query)
}

func (a *queryStoreAdapter) GetVersion(ctx context.Context, packageName string, version string) (internaldomain.SkillVersionDetail, error) {
	return a.store.GetVersion(ctx, packageName, version)
}

func (a *queryStoreAdapter) GetLatestResolvableVersion(ctx context.Context, packageName string) (internaldomain.SkillVersionDetail, error) {
	version, err := a.store.GetLatestResolvableVersion(ctx, packageName)
	if errors.Is(err, errNoResolvableVersion) {
		return internaldomain.SkillVersionDetail{}, internaldomain.ErrNoResolvableVersion
	}
	return version, err
}

func (a *queryStoreAdapter) ListGovernanceChanges(ctx context.Context, query internaldomain.GovernanceQuery) (internaldomain.GovernancePage, error) {
	return a.store.ListGovernanceChanges(ctx, query)
}

func (a *queryStoreAdapter) ListPackageVersions(ctx context.Context, packageName string) ([]internaldomain.SkillVersionDetail, error) {
	return a.store.ListPackageVersions(ctx, packageName)
}

func (a *queryStoreAdapter) ListPackageGovernance(ctx context.Context, packageName string, targetType internaldomain.GovernanceTargetType, version *string) ([]internaldomain.GovernanceRecord, error) {
	return a.store.ListPackageGovernance(ctx, packageName, GovernanceTargetType(targetType), version)
}
