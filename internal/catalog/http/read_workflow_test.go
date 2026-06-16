package http

import (
	"context"
	"reflect"
	"testing"

	catalog "skill_registry/internal/catalog/domain"
	"skill_registry/internal/platform/auth"
)

func TestReadModuleListsSkillsFromParsedQuery(t *testing.T) {
	service := &capturingCatalogReadService{
		response: catalog.SkillsPage{
			Items: []catalog.SkillSummary{
				{PackageName: "@platform.skill/pdf-skill"},
			},
		},
	}
	module := NewReadWorkflow(service)
	principal := auth.Principal{
		Role:  "reader",
		Actor: "reader",
	}

	response, err := module.ListSkills(context.Background(), principal, SkillsQuery{
		Query:          "pdf",
		PackageName:    "@platform.skill/pdf-skill",
		SourceType:     string(catalog.SourceTypeImported),
		IncludeRetired: "true",
		Sort:           "package_name_asc",
		PageSize:       "7",
		Cursor:         "cursor_12",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got := service.listSkillsCallCount(); got != 1 {
		t.Fatalf("expected catalog read service to be called once, got %d", got)
	}
	if !reflect.DeepEqual(service.query.AllowedPackages, principal.AllowedPackages) {
		t.Fatalf("unexpected allowed packages (-want +got):\nwant: %#v\ngot:  %#v", principal.AllowedPackages, service.query.AllowedPackages)
	}

	service.query.AllowedPackages = nil
	wantQuery := catalog.SkillQuery{
		Query:          "pdf",
		PackageName:    "@platform.skill/pdf-skill",
		SourceType:     string(catalog.SourceTypeImported),
		IncludeRetired: true,
		Sort:           "package_name_asc",
		PageSize:       7,
		Cursor:         12,
	}
	if !reflect.DeepEqual(service.query, wantQuery) {
		t.Fatalf("unexpected parsed query (-want +got):\nwant: %#v\ngot:  %#v", wantQuery, service.query)
	}
	if !reflect.DeepEqual(response, service.response) {
		t.Fatalf("expected response passthrough, got %#v", response)
	}
}

type capturingCatalogReadService struct {
	query    catalog.SkillQuery
	response catalog.SkillsPage
	calls    int
}

func (s *capturingCatalogReadService) ListSkills(_ context.Context, query catalog.SkillQuery) (catalog.SkillsPage, error) {
	s.calls++
	s.query = query
	return s.response, nil
}

func (s *capturingCatalogReadService) ListCuratedSkills(context.Context, catalog.CuratedSkillQuery) (catalog.CuratedSkillsPage, error) {
	return catalog.CuratedSkillsPage{}, nil
}

func (s *capturingCatalogReadService) GetSkill(context.Context, string) (catalog.SkillDetail, error) {
	return catalog.SkillDetail{}, nil
}

func (s *capturingCatalogReadService) ListVersions(context.Context, catalog.VersionQuery) (catalog.VersionsPage, error) {
	return catalog.VersionsPage{}, nil
}

func (s *capturingCatalogReadService) GetVersion(context.Context, string, string) (catalog.SkillVersionDetail, error) {
	return catalog.SkillVersionDetail{}, nil
}

func (s *capturingCatalogReadService) ResolveInstallTarget(context.Context, string, string) (catalog.InstallResolution, error) {
	return catalog.InstallResolution{}, nil
}

func (s *capturingCatalogReadService) listSkillsCallCount() int {
	return s.calls
}
