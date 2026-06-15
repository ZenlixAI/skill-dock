package catalogapi

import (
	"context"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/common"
	"skill_registry/business/catalog"
)

type catalogReadWorkflow interface {
	listSkills(ctx context.Context, principal auth.Principal, query catalog.SkillQuery) (catalog.SkillsPage, *catalog.Error)
	listCuratedSkills(ctx context.Context, principal auth.Principal, pageSize int, cursor int) (catalog.CuratedSkillsPage, *catalog.Error)
}

type catalogReadModule struct {
	repo catalog.ReadRepository
}

func (m *catalogReadModule) listSkills(ctx context.Context, principal auth.Principal, query catalog.SkillQuery) (catalog.SkillsPage, *catalog.Error) {
	query.AllowedPackages = append([]string(nil), principal.AllowedPackages...)
	response, err := m.repo.ListSkills(ctx, query)
	if err != nil {
		return catalog.SkillsPage{}, common.CatalogErrorFromError(err)
	}
	return response, nil
}

func (m *catalogReadModule) listCuratedSkills(ctx context.Context, principal auth.Principal, pageSize int, cursor int) (catalog.CuratedSkillsPage, *catalog.Error) {
	response, err := m.repo.ListCuratedSkills(ctx, catalog.CuratedSkillQuery{
		AllowedPackages: append([]string(nil), principal.AllowedPackages...),
		PageSize:        pageSize,
		Cursor:          cursor,
	})
	if err != nil {
		return catalog.CuratedSkillsPage{}, common.CatalogErrorFromError(err)
	}
	return response, nil
}
