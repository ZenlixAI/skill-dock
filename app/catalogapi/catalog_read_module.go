package catalogapi

import (
	"context"
	"strings"

	catalog "skill_registry/internal/catalog/domain"
	internalcataloghttp "skill_registry/internal/catalog/http"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
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
	listSkills(ctx context.Context, principal auth.Principal, query internalcataloghttp.SkillsQuery) (catalog.SkillsPage, *catalog.Error)
	listCuratedSkills(ctx context.Context, principal auth.Principal, pageSize int, cursor int) (catalog.CuratedSkillsPage, *catalog.Error)
	getSkill(ctx context.Context, packageName string) (catalog.SkillDetail, error)
	listVersions(ctx context.Context, query catalog.VersionQuery) (catalog.VersionsPage, error)
	getVersion(ctx context.Context, packageName string, version string) (catalog.SkillVersionDetail, error)
	resolveInstallTarget(ctx context.Context, packageName string, version string) (catalog.InstallResolution, error)
}

type catalogReadModule struct {
	service catalogReadService
}

func (m *catalogReadModule) listSkills(ctx context.Context, principal auth.Principal, query internalcataloghttp.SkillsQuery) (catalog.SkillsPage, *catalog.Error) {
	sourceType, sourceTypeErr := parseSourceTypeFilter(query.SourceType)
	if sourceTypeErr != nil {
		return catalog.SkillsPage{}, sourceTypeErr
	}
	includeRetired, includeRetiredErr := internalcataloghttp.ParseBooleanQueryParam(query.IncludeRetired, "include_retired", false)
	if includeRetiredErr != nil {
		return catalog.SkillsPage{}, includeRetiredErr
	}
	sortValue, sortErr := parseSkillsSort(query.Sort)
	if sortErr != nil {
		return catalog.SkillsPage{}, sortErr
	}
	pageSize, pageSizeErr := internalcataloghttp.ParsePageSizeValue(query.PageSize)
	if pageSizeErr != nil {
		return catalog.SkillsPage{}, pageSizeErr
	}
	cursor, cursorErr := internalcataloghttp.ParseCursorValue(query.Cursor)
	if cursorErr != nil {
		return catalog.SkillsPage{}, cursorErr
	}

	skillQuery := catalog.SkillQuery{
		Query:          query.Query,
		PackageName:    query.PackageName,
		SourceType:     sourceType,
		IncludeRetired: includeRetired,
		Sort:           sortValue,
		PageSize:       pageSize,
		Cursor:         cursor,
	}
	skillQuery.AllowedPackages = append([]string(nil), principal.AllowedPackages...)

	response, err := m.service.ListSkills(ctx, skillQuery)
	if err != nil {
		return catalog.SkillsPage{}, platformhttp.CatalogErrorFromError(err)
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
		return catalog.CuratedSkillsPage{}, platformhttp.CatalogErrorFromError(err)
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
	case "", string(catalog.SourceTypeNative), string(catalog.SourceTypeImported), string(catalog.SourceTypePlatformCreated):
		return value, nil
	default:
		return "", catalog.NewError(400, "invalid_request", "source_type is invalid.", map[string]any{"source_type": value})
	}
}
