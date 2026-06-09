package catalogapi

import (
	"context"
	"net/url"

	"skill_registry/app/auth"
	"skill_registry/business/catalog"
)

type readAPIWorkflow interface {
	listSkills(ctx context.Context, principal auth.Principal, values url.Values) (catalog.SkillsPage, *catalog.Error)
}

type readAPIModule struct {
	catalogReadModule catalogReadWorkflow
}

func (m *readAPIModule) listSkills(ctx context.Context, principal auth.Principal, values url.Values) (catalog.SkillsPage, *catalog.Error) {
	sourceType, sourceTypeErr := parseSourceTypeFilter(values.Get("source_type"))
	if sourceTypeErr != nil {
		return catalog.SkillsPage{}, sourceTypeErr
	}
	includeRetired, includeRetiredErr := parseBooleanQueryParam(values.Get("include_retired"), "include_retired", false)
	if includeRetiredErr != nil {
		return catalog.SkillsPage{}, includeRetiredErr
	}
	sortValue, sortErr := parseSkillsSort(values.Get("sort"))
	if sortErr != nil {
		return catalog.SkillsPage{}, sortErr
	}
	pageSize, pageSizeErr := parsePageSize(values)
	if pageSizeErr != nil {
		return catalog.SkillsPage{}, pageSizeErr
	}
	cursor, cursorErr := parseCursor(values)
	if cursorErr != nil {
		return catalog.SkillsPage{}, cursorErr
	}

	return m.catalogReadModule.listSkills(ctx, principal, catalog.SkillQuery{
		Query:          values.Get("q"),
		PackageName:    values.Get("package_name"),
		SourceType:     sourceType,
		IncludeRetired: includeRetired,
		Sort:           sortValue,
		PageSize:       pageSize,
		Cursor:         cursor,
	})
}
