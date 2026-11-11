package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	catalog "skill_registry/internal/catalog/domain"
	cataloginfra "skill_registry/internal/catalog/infra"
)

type fakeExternalSkillSource struct {
	searchPage catalog.ExternalSkillsPage
	listPage   catalog.ExternalSkillsPage
	detail     catalog.ExternalSkillDetail
	searchErr  error
	listErr    error
	detailErr  error

	searchCalls int
	listCalls   int
	detailCalls int
	lastSearch  catalog.ExternalSearchQuery
	lastList    catalog.ExternalBrowseQuery
}

func (f *fakeExternalSkillSource) SearchSkills(_ context.Context, query catalog.ExternalSearchQuery) (catalog.ExternalSkillsPage, error) {
	f.searchCalls++
	f.lastSearch = query
	return f.searchPage, f.searchErr
}

func (f *fakeExternalSkillSource) ListSkills(_ context.Context, query catalog.ExternalBrowseQuery) (catalog.ExternalSkillsPage, error) {
	f.listCalls++
	f.lastList = query
	return f.listPage, f.listErr
}

func (f *fakeExternalSkillSource) GetSkill(_ context.Context, source string, skillID string) (catalog.ExternalSkillDetail, error) {
	f.detailCalls++
	return f.detail, f.detailErr
}

func TestPublicRegistryCapabilities(t *testing.T) {
	server := NewServer(ServerConfig{Repository: cataloginfra.NewMemoryRepository(cataloginfra.SampleData())})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/public/registry/capabilities", nil)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload publicRegistryCapabilities
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.ExternalSearch.Enabled {
		t.Fatal("expected external search to be disabled")
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected CORS header, got %q", got)
	}
}

func TestPublicRegistrySkillsNeverMergesExternalResults(t *testing.T) {
	external := &fakeExternalSkillSource{
		searchPage: catalog.ExternalSkillsPage{Items: []catalog.ExternalSkill{{ID: "owner/repo/pdf", Name: "pdf"}}},
	}
	server := NewServer(ServerConfig{
		Repository:            cataloginfra.NewMemoryRepository(cataloginfra.SampleData()),
		ExternalSkillSearcher: external,
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/public/registry/skills?q=pdf&include_external=true&source=all", nil)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload catalog.SkillsPage
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Items) == 0 {
		t.Fatal("expected catalog results")
	}
	for _, item := range payload.Items {
		if item.SourceType == catalog.SourceTypeExternal {
			t.Fatalf("public registry returned external item %#v", item)
		}
	}
	if external.searchCalls != 0 {
		t.Fatalf("expected no external search, got %d calls", external.searchCalls)
	}
}

func TestPublicRegistryExternalSearch(t *testing.T) {
	external := &fakeExternalSkillSource{
		searchPage: catalog.ExternalSkillsPage{Items: []catalog.ExternalSkill{{
			ID: "vercel-labs/agent-skills/vercel-react-best-practices", Source: "vercel-labs/agent-skills", SkillID: "vercel-react-best-practices", Name: "vercel-react-best-practices", Installs: 885584,
		}}},
	}
	server := NewServer(ServerConfig{
		Repository:            cataloginfra.NewMemoryRepository(cataloginfra.SampleData()),
		ExternalSkillSearcher: external,
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/public/registry/external/skills?q=react&limit=5", nil)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if external.searchCalls != 1 || external.lastSearch.Query != "react" || external.lastSearch.PageSize != 5 {
		t.Fatalf("unexpected search call %#v", external.lastSearch)
	}
	var payload catalog.ExternalSkillsPage
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(payload.Items) != 1 || payload.Items[0].Installs != 885584 {
		t.Fatalf("unexpected payload %#v", payload)
	}
}

func TestPublicRegistryExternalBrowse(t *testing.T) {
	external := &fakeExternalSkillSource{
		listPage: catalog.ExternalSkillsPage{Items: []catalog.ExternalSkill{{ID: "owner/repo/one", Name: "one"}}, Total: 10, Page: 2},
	}
	server := NewServer(ServerConfig{
		Repository:            cataloginfra.NewMemoryRepository(cataloginfra.SampleData()),
		ExternalSkillSearcher: external,
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/public/registry/external/skills?view=trending&page=2&limit=3", nil)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if external.listCalls != 1 || external.lastList.View != "trending" || external.lastList.Page != 2 || external.lastList.PageSize != 3 {
		t.Fatalf("unexpected list call %#v", external.lastList)
	}
}

func TestPublicRegistryExternalDetail(t *testing.T) {
	external := &fakeExternalSkillSource{
		detail: catalog.ExternalSkillDetail{
			ExternalSkill: catalog.ExternalSkill{ID: "owner/repo/one", Source: "owner/repo", SkillID: "one", Name: "one", Installs: 4},
			Description:   "Detail description",
			SummaryHTML:   "<p>Summary</p>",
			ContentHTML:   "<p>Content</p>",
		},
	}
	server := NewServer(ServerConfig{
		Repository:            cataloginfra.NewMemoryRepository(cataloginfra.SampleData()),
		ExternalSkillSearcher: external,
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/public/registry/external/skill?source=owner/repo&skillId=one", nil)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var payload catalog.ExternalSkillDetail
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Description != "Detail description" || payload.Installs != 4 {
		t.Fatalf("unexpected payload %#v", payload)
	}
}

func TestPublicRegistryExternalDisabled(t *testing.T) {
	server := NewServer(ServerConfig{Repository: cataloginfra.NewMemoryRepository(cataloginfra.SampleData())})
	for _, path := range []string{
		"/api/v1/public/registry/external/skills?q=react",
		"/api/v1/public/registry/external/skill?source=owner/repo&skillId=one",
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		server.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s: expected 404, got %d", path, recorder.Code)
		}
	}
}

func TestPublicRegistryPreflight(t *testing.T) {
	server := NewServer(ServerConfig{Repository: cataloginfra.NewMemoryRepository(cataloginfra.SampleData())})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/public/registry/skills", nil)
	server.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("expected CORS methods header")
	}
}

func TestPublicRegistrySkillsShAliases(t *testing.T) {
	external := &fakeExternalSkillSource{
		searchPage: catalog.ExternalSkillsPage{Items: []catalog.ExternalSkill{{ID: "owner/repo/one", Name: "one"}}},
		listPage:   catalog.ExternalSkillsPage{Items: []catalog.ExternalSkill{{ID: "owner/repo/two", Name: "two"}}, Total: 2, Page: 2},
	}
	server := NewServer(ServerConfig{
		Repository:            cataloginfra.NewMemoryRepository(cataloginfra.SampleData()),
		ExternalSkillSearcher: external,
	})

	searchRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(searchRecorder, httptest.NewRequest(http.MethodGet, "/api/skills-sh/search?q=react&limit=5", nil))
	if searchRecorder.Code != http.StatusOK {
		t.Fatalf("search: expected 200, got %d: %s", searchRecorder.Code, searchRecorder.Body.String())
	}
	if external.searchCalls != 1 || external.lastSearch.Query != "react" || external.lastSearch.PageSize != 5 {
		t.Fatalf("unexpected search call %#v", external.lastSearch)
	}

	browseRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(browseRecorder, httptest.NewRequest(http.MethodGet, "/api/skills-sh/browse?view=trending&page=2&limit=3", nil))
	if browseRecorder.Code != http.StatusOK {
		t.Fatalf("browse: expected 200, got %d: %s", browseRecorder.Code, browseRecorder.Body.String())
	}
	if external.listCalls != 1 || external.lastList.View != "trending" || external.lastList.Page != 2 || external.lastList.PageSize != 3 {
		t.Fatalf("unexpected browse call %#v", external.lastList)
	}
}
