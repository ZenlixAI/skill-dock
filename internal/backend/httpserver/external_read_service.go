package httpserver

import (
	"context"
	"strings"

	catalog "skill_registry/internal/catalog/domain"
)

type externalReadService struct {
	base     catalog.ReadRepository
	external catalog.ExternalSkillSearcher
}

func newExternalReadService(base catalog.ReadRepository, external catalog.ExternalSkillSearcher) catalog.ReadRepository {
	if base == nil || external == nil {
		return base
	}
	return &externalReadService{base: base, external: external}
}

func (s *externalReadService) ListSkills(ctx context.Context, query catalog.SkillQuery) (catalog.SkillsPage, error) {
	basePage, err := s.base.ListSkills(ctx, query)
	if err != nil {
		return catalog.SkillsPage{}, err
	}

	if !shouldIncludeExternalSkills(query) {
		return basePage, nil
	}

	externalPage, externalErr := s.external.SearchSkills(ctx, catalog.ExternalSearchQuery{
		Query:    query.Query,
		PageSize: query.PageSize,
		Registry: query.Registry,
		Scope:    query.Scope,
		Origin:   query.Origin,
	})
	if externalErr != nil {
		if len(basePage.Items) > 0 {
			return basePage, nil
		}
		return catalog.SkillsPage{}, externalErr
	}

	return mergeSkillsPages(basePage, externalPage, query), nil
}

func (s *externalReadService) ListCuratedSkills(ctx context.Context, query catalog.CuratedSkillQuery) (catalog.CuratedSkillsPage, error) {
	return s.base.ListCuratedSkills(ctx, query)
}

func (s *externalReadService) GetSkill(ctx context.Context, packageName string) (catalog.SkillDetail, error) {
	return s.base.GetSkill(ctx, packageName)
}

func (s *externalReadService) ListVersions(ctx context.Context, query catalog.VersionQuery) (catalog.VersionsPage, error) {
	return s.base.ListVersions(ctx, query)
}

func (s *externalReadService) GetVersion(ctx context.Context, packageName string, version string) (catalog.SkillVersionDetail, error) {
	return s.base.GetVersion(ctx, packageName, version)
}

func (s *externalReadService) ListGovernanceChanges(ctx context.Context, query catalog.GovernanceQuery) (catalog.GovernancePage, error) {
	return s.base.ListGovernanceChanges(ctx, query)
}

func (s *externalReadService) ResolveInstallTarget(ctx context.Context, packageName string, version string) (catalog.InstallResolution, error) {
	return s.base.ResolveInstallTarget(ctx, packageName, version)
}

func shouldIncludeExternalSkills(query catalog.SkillQuery) bool {
	if query.IncludeExternal {
		return true
	}
	if strings.EqualFold(query.Source, "all") || strings.EqualFold(query.Source, "skills.sh") {
		return true
	}
	return strings.EqualFold(query.SourceType, string(catalog.SourceTypeExternal))
}

func mergeSkillsPages(basePage catalog.SkillsPage, externalPage catalog.SkillsPage, query catalog.SkillQuery) catalog.SkillsPage {
	items := make([]catalog.SkillSummary, 0, len(basePage.Items)+len(externalPage.Items))
	seen := map[string]struct{}{}
	appendItem := func(item catalog.SkillSummary) {
		if !matchesExternalSkillQuery(query, item) {
			return
		}
		key := item.PackageName + "\x00" + strings.TrimSpace(item.SourceKind)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		items = append(items, item)
	}

	for _, item := range basePage.Items {
		appendItem(item)
	}
	for _, item := range externalPage.Items {
		appendItem(item)
	}

	return catalog.SkillsPage{
		Items: items,
		PageInfo: catalog.PageInfo{
			HasNextPage: basePage.PageInfo.HasNextPage,
			NextCursor:  basePage.PageInfo.NextCursor,
		},
	}
}

func matchesExternalSkillQuery(query catalog.SkillQuery, item catalog.SkillSummary) bool {
	if query.SourceType != "" && string(item.SourceType) != query.SourceType {
		return false
	}
	if query.PackageName != "" && item.PackageName != query.PackageName {
		return false
	}
	if query.Origin != "" {
		origin := item.OriginRegistryURL
		if origin == "" {
			origin = item.OriginRepositoryURL
		}
		if !strings.Contains(origin, query.Origin) {
			return false
		}
	}
	if query.Scope != "" {
		if !strings.HasPrefix(item.PackageName, "@"+query.Scope+"/") && !strings.HasPrefix(item.PackageName, query.Scope+"/") {
			return false
		}
	}
	return true
}
