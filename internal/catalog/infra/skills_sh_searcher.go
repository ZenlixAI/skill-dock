package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	catalog "skill_registry/internal/catalog/domain"
)

type skillsShSearchSource struct {
	baseURL string
	client  *http.Client
}

func NewSkillsShSearchSource(baseURL string, client *http.Client) catalog.ExternalSkillSearcher {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		trimmed = "https://skills.sh"
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &skillsShSearchSource{baseURL: trimmed, client: client}
}

func (s *skillsShSearchSource) SearchSkills(ctx context.Context, query catalog.ExternalSearchQuery) (catalog.SkillsPage, error) {
	endpoint, err := url.Parse(s.baseURL)
	if err != nil {
		return catalog.SkillsPage{}, err
	}
	endpoint.Path = "/-/v1/search"
	params := endpoint.Query()
	params.Set("text", query.Query)
	params.Set("size", fmt.Sprintf("%d", max(query.PageSize, 10)))
	if query.Registry != "" {
		params.Set("registry", query.Registry)
	}
	if query.Scope != "" {
		params.Set("scope", query.Scope)
	}
	if query.Origin != "" {
		params.Set("origin", query.Origin)
	}
	endpoint.RawQuery = params.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return catalog.SkillsPage{}, err
	}
	request.Header.Set("accept", "application/json")

	response, err := s.client.Do(request)
	if err != nil {
		return catalog.SkillsPage{}, err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return catalog.SkillsPage{}, fmt.Errorf("skills.sh search failed: %s", response.Status)
	}

	var payload struct {
		Objects []struct {
			Package struct {
				Name        string `json:"name"`
				Version     string `json:"version"`
				Description string `json:"description"`
			} `json:"package"`
		} `json:"objects"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return catalog.SkillsPage{}, err
	}

	items := make([]catalog.SkillSummary, 0, len(payload.Objects))
	for _, entry := range payload.Objects {
		pkg := entry.Package
		if pkg.Name == "" {
			continue
		}
		items = append(items, catalog.SkillSummary{
			PackageName:         pkg.Name,
			DisplayName:         pkg.Name,
			Description:         pkg.Description,
			SourceType:          catalog.SourceTypeExternal,
			Status:              catalog.SkillStatusActive,
			LatestVersion:       stringPtrValue(pkg.Version),
			LatestVersionStatus: stringPtr("published"),
			SourceKind:          "skills.sh",
			OriginRegistryURL:   s.baseURL,
			LearnMoreURL:        fmt.Sprintf("%s/%s", s.baseURL, url.PathEscape(pkg.Name)),
			Installable:         false,
			Hosted:              false,
			Mirrored:            false,
		})
	}

	return catalog.SkillsPage{Items: items, PageInfo: catalog.PageInfo{HasNextPage: false, NextCursor: nil}}, nil
}

func stringPtrValue(value string) *string {
	return &value
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
