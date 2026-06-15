package catalogapi

import (
	"context"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/common"
	"skill_registry/business/catalog"
)

type catalogReadService interface {
	ListSkills(ctx context.Context, query catalog.SkillQuery) (catalog.SkillsPage, error)
	ListCuratedSkills(ctx context.Context, query catalog.CuratedSkillQuery) (catalog.CuratedSkillsPage, error)
	GetSkill(ctx context.Context, packageName string) (catalog.SkillDetail, error)
	ListVersions(ctx context.Context, query catalog.VersionQuery) (catalog.VersionsPage, error)
	GetVersion(ctx context.Context, packageName string, version string) (catalog.SkillVersionDetail, error)
	ResolveInstallTarget(ctx context.Context, packageName string, version string) (catalog.InstallResolution, error)
}

type catalogReadWorkflow interface {
	listSkills(ctx context.Context, principal auth.Principal, query catalog.SkillQuery) (catalog.SkillsPage, *catalog.Error)
	listCuratedSkills(ctx context.Context, principal auth.Principal, pageSize int, cursor int) (catalog.CuratedSkillsPage, *catalog.Error)
	getSkill(ctx context.Context, packageName string) (catalog.SkillDetail, error)
	listVersions(ctx context.Context, query catalog.VersionQuery) (catalog.VersionsPage, error)
	getVersion(ctx context.Context, packageName string, version string) (catalog.SkillVersionDetail, error)
	resolveInstallTarget(ctx context.Context, packageName string, version string) (catalog.InstallResolution, error)
}

type catalogReadModule struct {
	service catalogReadService
}

func (m *catalogReadModule) listSkills(ctx context.Context, principal auth.Principal, query catalog.SkillQuery) (catalog.SkillsPage, *catalog.Error) {
	query.AllowedPackages = append([]string(nil), principal.AllowedPackages...)
	response, err := m.service.ListSkills(ctx, query)
	if err != nil {
		return catalog.SkillsPage{}, common.CatalogErrorFromError(err)
	}
	return response, nil
}

func (m *catalogReadModule) listCuratedSkills(ctx context.Context, principal auth.Principal, pageSize int, cursor int) (catalog.CuratedSkillsPage, *catalog.Error) {
	response, err := m.service.ListCuratedSkills(ctx, catalog.CuratedSkillQuery{
		AllowedPackages: append([]string(nil), principal.AllowedPackages...),
		PageSize:        pageSize,
		Cursor:          cursor,
	})
	if err != nil {
		return catalog.CuratedSkillsPage{}, common.CatalogErrorFromError(err)
	}
	return response, nil
}

func (m *catalogReadModule) getSkill(ctx context.Context, packageName string) (catalog.SkillDetail, error) {
	return m.service.GetSkill(ctx, packageName)
}

func (m *catalogReadModule) listVersions(ctx context.Context, query catalog.VersionQuery) (catalog.VersionsPage, error) {
	return m.service.ListVersions(ctx, query)
}

func (m *catalogReadModule) getVersion(ctx context.Context, packageName string, version string) (catalog.SkillVersionDetail, error) {
	return m.service.GetVersion(ctx, packageName, version)
}

func (m *catalogReadModule) resolveInstallTarget(ctx context.Context, packageName string, version string) (catalog.InstallResolution, error) {
	return m.service.ResolveInstallTarget(ctx, packageName, version)
}
