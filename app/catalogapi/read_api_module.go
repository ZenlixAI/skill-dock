package catalogapi

import (
	"context"
	"strings"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/binding"
	"skill_registry/app/catalogapi/common"
	"skill_registry/business/catalog"
)

type readAPIWorkflow interface {
	listSkills(ctx context.Context, principal auth.Principal, query binding.SkillsQuery) (catalog.SkillsPage, *catalog.Error)
}

type readAPIModule struct {
	catalogReadModule catalogReadWorkflow
}

func (m *readAPIModule) listSkills(ctx context.Context, principal auth.Principal, query binding.SkillsQuery) (catalog.SkillsPage, *catalog.Error) {
	sourceType, sourceTypeErr := parseSourceTypeFilter(query.SourceType)
	if sourceTypeErr != nil {
		return catalog.SkillsPage{}, sourceTypeErr
	}
	includeRetired, includeRetiredErr := common.ParseBooleanQueryParam(query.IncludeRetired, "include_retired", false)
	if includeRetiredErr != nil {
		return catalog.SkillsPage{}, includeRetiredErr
	}
	sortValue, sortErr := parseSkillsSort(query.Sort)
	if sortErr != nil {
		return catalog.SkillsPage{}, sortErr
	}
	pageSize, pageSizeErr := common.ParsePageSizeValue(query.PageSize)
	if pageSizeErr != nil {
		return catalog.SkillsPage{}, pageSizeErr
	}
	cursor, cursorErr := common.ParseCursorValue(query.Cursor)
	if cursorErr != nil {
		return catalog.SkillsPage{}, cursorErr
	}

	return m.catalogReadModule.listSkills(ctx, principal, catalog.SkillQuery{
		Query:          query.Query,
		PackageName:    query.PackageName,
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
