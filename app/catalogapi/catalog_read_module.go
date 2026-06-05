package catalogapi

import (
	"context"

	"skill_registry/app/auth"
	"skill_registry/business/catalog"
)

type catalogReadWorkflow interface {
	listSkills(ctx context.Context, principal auth.Principal, query catalog.SkillQuery) (catalog.SkillsPage, *catalog.Error)
	listCuratedSkills(ctx context.Context, principal auth.Principal, pageSize int, cursor int) (catalog.CuratedSkillsPage, *catalog.Error)
}

type catalogReadModule struct {
	repo catalog.Repository
}

func (m *catalogReadModule) listSkills(ctx context.Context, principal auth.Principal, query catalog.SkillQuery) (catalog.SkillsPage, *catalog.Error) {
	response, err := m.repo.ListSkills(ctx, query)
	if err != nil {
		return catalog.SkillsPage{}, catalogErrorFromError(err)
	}
	response.Items = auth.FilterSkillSummariesByPrincipal(response.Items, principal)
	return response, nil
}

func (m *catalogReadModule) listCuratedSkills(ctx context.Context, principal auth.Principal, pageSize int, cursor int) (catalog.CuratedSkillsPage, *catalog.Error) {
	response, err := m.repo.ListCuratedSkills(ctx, pageSize, cursor)
	if err != nil {
		return catalog.CuratedSkillsPage{}, catalogErrorFromError(err)
	}
	response.Items = auth.FilterCuratedSkillSummariesByPrincipal(response.Items, principal)
	return response, nil
}
