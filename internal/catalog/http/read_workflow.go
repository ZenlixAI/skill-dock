package http

import (
	"context"
	"strings"

	catalog "skill_registry/internal/catalog/domain"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
)

type ReadModule struct {
	service catalog.ReadRepository
}

func NewReadWorkflow(service catalog.ReadRepository) *ReadModule {
	return &ReadModule{service: service}
}

func (m *ReadModule) ListSkills(ctx context.Context, principal auth.Principal, query SkillsQuery) (catalog.SkillsPage, *catalog.Error) {
	includeRetired, includeRetiredErr := ParseBooleanQueryParam(query.IncludeRetired, "include_retired", false)
	if includeRetiredErr != nil {
		return catalog.SkillsPage{}, includeRetiredErr
	}
	sortValue, sortErr := parseSkillsSort(query.Sort)
	if sortErr != nil {
		return catalog.SkillsPage{}, sortErr
	}
	pageSize, pageSizeErr := ParsePageSizeValue(query.PageSize)
	if pageSizeErr != nil {
		return catalog.SkillsPage{}, pageSizeErr
	}
	cursor, cursorErr := ParseCursorValue(query.Cursor)
	if cursorErr != nil {
		return catalog.SkillsPage{}, cursorErr
	}

	includeExternal, includeExternalErr := ParseBooleanQueryParam(query.IncludeExternal, "include_external", false)
	if includeExternalErr != nil {
		return catalog.SkillsPage{}, includeExternalErr
	}
	sourceTypeFilter, sourceTypeErr := parseSourceTypeFilter(query.SourceType)
	if sourceTypeErr != nil {
		return catalog.SkillsPage{}, sourceTypeErr
	}
	source := strings.TrimSpace(query.Source)
	registry := strings.TrimSpace(query.Registry)
	scope := strings.TrimSpace(query.Scope)
	origin := strings.TrimSpace(query.Origin)
	if includeExternal && source == "catalog" {
		source = "all"
	}

	skillQuery := catalog.SkillQuery{
		Query:           query.Query,
		PackageName:     query.PackageName,
		SourceType:      sourceTypeFilter,
		IncludeRetired:  includeRetired,
		IncludeExternal: includeExternal,
		Source:          source,
		Registry:        registry,
		Scope:           scope,
		Origin:          origin,
		Sort:            sortValue,
		PageSize:        pageSize,
		Cursor:          cursor,
	}
	skillQuery.AllowedPackages = append([]string(nil), principal.AllowedPackages...)

	response, err := m.service.ListSkills(ctx, skillQuery)
	if err != nil {
		return catalog.SkillsPage{}, platformhttp.CatalogErrorFromError(err)
	}
	return response, nil
}

func (m *ReadModule) ListCuratedSkills(ctx context.Context, principal auth.Principal, pageSize int, cursor int) (catalog.CuratedSkillsPage, *catalog.Error) {
	response, err := m.service.ListCuratedSkills(ctx, catalog.CuratedSkillQuery{
		AllowedPackages: append([]string(nil), principal.AllowedPackages...),
		PageSize:        pageSize,
		Cursor:          cursor,
	})
	if err != nil {
		return catalog.CuratedSkillsPage{}, platformhttp.CatalogErrorFromError(err)
	}
	return response, nil
}

func (m *ReadModule) GetSkill(ctx context.Context, packageName string) (catalog.SkillDetail, error) {
	return m.service.GetSkill(ctx, packageName)
}

func (m *ReadModule) ListVersions(ctx context.Context, query catalog.VersionQuery) (catalog.VersionsPage, error) {
	return m.service.ListVersions(ctx, query)
}

func (m *ReadModule) GetVersion(ctx context.Context, packageName string, version string) (catalog.SkillVersionDetail, error) {
	return m.service.GetVersion(ctx, packageName, version)
}

func (m *ReadModule) ResolveInstallTarget(ctx context.Context, packageName string, version string) (catalog.InstallResolution, error) {
	return m.service.ResolveInstallTarget(ctx, packageName, version)
}

func parseSkillsSort(raw string) (string, *catalog.Error) {
	value := strings.TrimSpace(raw)
	switch value {
	case "", "relevance", "updated_at_desc", "package_name_asc":
		return value, nil
	default:
		return "", catalog.NewError(400, "invalid_request", "sort is invalid.", map[string]any{"sort": value})
	}
}

func parseSourceTypeFilter(raw string) (string, *catalog.Error) {
	value := strings.TrimSpace(raw)
	switch value {
	case "", string(catalog.SourceTypeNative), string(catalog.SourceTypeImported), string(catalog.SourceTypeExternal), string(catalog.SourceTypePlatformCreated):
		return value, nil
	default:
		return "", catalog.NewError(400, "invalid_request", "source_type is invalid.", map[string]any{"source_type": value})
	}
}
