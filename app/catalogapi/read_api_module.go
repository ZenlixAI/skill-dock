package catalogapi

import (
	"context"
	"net/url"
	"strings"

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
