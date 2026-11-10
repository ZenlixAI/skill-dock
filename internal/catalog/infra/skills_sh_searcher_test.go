package infra

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	catalog "skill_registry/internal/catalog/domain"
)

func TestSkillsShSearchSkills(t *testing.T) {
	var gotQuery string
	var gotLimit string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/search" {
			t.Fatalf("unexpected path %s", request.URL.Path)
		}
		gotQuery = request.URL.Query().Get("q")
		gotLimit = request.URL.Query().Get("limit")
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"skills":[{"id":"vercel-labs/agent-skills/vercel-react-best-practices","source":"vercel-labs/agent-skills","skillId":"vercel-react-best-practices","name":"vercel-react-best-practices","installs":885584}]}`))
	}))
	defer server.Close()

	source := NewSkillsShSearchSource(server.URL, server.Client())
	page, err := source.SearchSkills(context.Background(), catalog.ExternalSearchQuery{Query: "react", PageSize: 7})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if gotQuery != "react" || gotLimit != "7" {
		t.Fatalf("unexpected query params q=%q limit=%q", gotQuery, gotLimit)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(page.Items))
	}
	item := page.Items[0]
	if item.ID != "vercel-labs/agent-skills/vercel-react-best-practices" || item.Name != "vercel-react-best-practices" || item.Installs != 885584 {
		t.Fatalf("unexpected item %#v", item)
	}
	if item.LearnMoreURL != server.URL+"/vercel-labs/agent-skills/vercel-react-best-practices" {
		t.Fatalf("unexpected learn more url %q", item.LearnMoreURL)
	}
}

func TestSkillsShSearchSkipsShortQueries(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()

	source := NewSkillsShSearchSource(server.URL, server.Client())
	page, err := source.SearchSkills(context.Background(), catalog.ExternalSearchQuery{Query: "r", PageSize: 10})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if called {
		t.Fatal("expected short query to skip the remote request")
	}
	if len(page.Items) != 0 {
		t.Fatalf("expected no items, got %d", len(page.Items))
	}
}

func TestSkillsShListSkills(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/skills/all-time/2" {
			t.Fatalf("unexpected path %s", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
			"skills":[
				{"source":"owner/one","skillId":"one","name":"one","installs":10,"weeklyInstalls":[1,2,3],"isOfficial":true},
				{"source":"owner/two","skillId":"two","name":"two","installs":9},
				{"source":"owner/three","skillId":"three","name":"three","installs":8}
			],
			"total":42,
			"hasMore":true,
			"page":2
		}`))
	}))
	defer server.Close()

	source := NewSkillsShSearchSource(server.URL, server.Client())
	page, err := source.ListSkills(context.Background(), catalog.ExternalBrowseQuery{View: "all-time", Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(page.Items))
	}
	if page.Total != 42 || page.Page != 2 || !page.PageInfo.HasNextPage {
		t.Fatalf("unexpected page metadata %#v", page)
	}
	if !page.Items[0].IsOfficial || len(page.Items[0].WeeklyInstalls) != 3 {
		t.Fatalf("unexpected first item %#v", page.Items[0])
	}
}

func TestSkillsShGetSkill(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/vercel-labs/agent-skills/vercel-react-best-practices" {
			t.Fatalf("unexpected path %s", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "text/html")
		_, _ = writer.Write([]byte(skillsShDetailFixture))
	}))
	defer server.Close()

	source := NewSkillsShSearchSource(server.URL, server.Client())
	detail, err := source.GetSkill(context.Background(), "vercel-labs/agent-skills", "vercel-react-best-practices")
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if detail.Name != "vercel-react-best-practices" || detail.DisplayName != "Vercel React Best Practices" {
		t.Fatalf("unexpected names %#v", detail)
	}
	if detail.Description != "Full description from JSON-LD." {
		t.Fatalf("unexpected description %q", detail.Description)
	}
	if detail.Installs != 886530 {
		t.Fatalf("unexpected installs %d", detail.Installs)
	}
	if detail.InstallCommand != "skills add vercel-labs/agent-skills --skill vercel-react-best-practices" {
		t.Fatalf("unexpected install command %q", detail.InstallCommand)
	}
	if len(detail.Topics) != 2 || detail.Topics[0] != "React" || detail.Topics[1] != "Next.js" {
		t.Fatalf("unexpected topics %#v", detail.Topics)
	}
	if len(detail.WeeklyInstalls) != 3 || detail.WeeklyInstalls[0] != 19251 {
		t.Fatalf("unexpected weekly installs %#v", detail.WeeklyInstalls)
	}
	if !detail.IsOfficial {
		t.Fatal("expected official badge")
	}
	if strings.Contains(detail.SummaryHTML, "<script") || !strings.Contains(detail.SummaryHTML, "<strong>Summary") {
		t.Fatalf("unexpected summary html %q", detail.SummaryHTML)
	}
	if strings.Contains(detail.ContentHTML, "onclick") ||
		strings.Contains(detail.ContentHTML, "javascript:") ||
		strings.Contains(detail.ContentHTML, "<img") ||
		strings.Contains(detail.ContentHTML, "onerror") ||
		!strings.Contains(detail.ContentHTML, `href="https://www.skills.sh/topic/react"`) {
		t.Fatalf("unexpected content html %q", detail.ContentHTML)
	}
	if len(detail.RelatedSkills) != 1 || detail.RelatedSkills[0].SkillID != "related-skill" {
		t.Fatalf("unexpected related skills %#v", detail.RelatedSkills)
	}
}

func TestSkillsShGetSkillRejectsInvalidReference(t *testing.T) {
	source := NewSkillsShSearchSource("", nil)
	if _, err := source.GetSkill(context.Background(), "../../etc", "passwd"); err == nil {
		t.Fatal("expected invalid source error")
	}
	if _, err := source.GetSkill(context.Background(), "owner/repo", "../x"); err == nil {
		t.Fatal("expected invalid skill id error")
	}
}

const skillsShDetailFixture = `<!DOCTYPE html>
<html>
<head>
<title>vercel-react-best-practices — vercel-labs/agent-skills</title>
<meta name="description" content="Short description." />
</head>
<body>
<main>
<nav aria-label="Breadcrumb"><a href="/">skills</a><a href="/vercel-labs">vercel-labs</a><a href="/vercel-labs/agent-skills">agent-skills</a></nav>
<h1>vercel-react-best-practices</h1>
<div><a href="/topic/react">React</a><a href="/topic/nextjs">Next.js</a></div>
<div class="uppercase">Installation</div>
<code>$ npx skills add https://github.com/vercel-labs/agent-skills --skill vercel-react-best-practices</code>
<div class="uppercase">Summary</div>
<div class="prose prose-invert"><p><strong>Summary text</strong></p><ul><li><a href="/topic/react">topic link</a></li></ul></div>
<div class="uppercase">SKILL.md</div>
<div class="prose prose-invert"><h1>Vercel React Best Practices</h1><p>The <a href="/topic/react" onclick="alert(1)">guide</a> and <a href="javascript:alert(1)">bad link</a>.</p><img src="x" onerror="alert(1)" /><script>alert(1)</script></div>
<div class="uppercase">Related skills</div>
<section><a href="/owner/repo/related-skill"><div><h3>related-skill</h3><p>Related description</p></div></a></section>
<div>Installs</div>
<svg aria-label="Weekly installs: 19,251, 18,945, 21,814"></svg>
<svg aria-label="Verified organization on GitHub"></svg>
</main>
<script type="application/ld+json">{"@context":"https://schema.org","@type":"SoftwareApplication","name":"vercel-react-best-practices","description":"Full description from JSON-LD.","publisher":{"@type":"Organization","name":"vercel-labs"},"interactionStatistic":{"@type":"InteractionCounter","interactionType":"https://schema.org/InstallAction","userInteractionCount":886530}}</script>
</body>
</html>`

func TestSkillsShGetSkillUsesFullRSCContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("RSC") == "1" {
			writer.Header().Set("Content-Type", "text/x-component")
			_, _ = writer.Write([]byte("1:T60,<p>RSC rest content</p>\n2:[\"$\",\"div\",null,{\"previewHtml\":\"<p>RSC preview content</p>\",\"restHtml\":\"$1\",\"proseClassName\":\"prose\"}]\n"))
			return
		}
		writer.Header().Set("Content-Type", "text/html")
		_, _ = writer.Write([]byte(skillsShDetailFixture))
	}))
	defer server.Close()

	source := NewSkillsShSearchSource(server.URL, server.Client())
	detail, err := source.GetSkill(context.Background(), "vercel-labs/agent-skills", "vercel-react-best-practices")
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if !strings.Contains(detail.ContentHTML, "RSC preview content") || !strings.Contains(detail.ContentHTML, "RSC rest content") {
		t.Fatalf("expected full RSC content, got %q", detail.ContentHTML)
	}
	if strings.Contains(detail.ContentHTML, "The <a") {
		t.Fatalf("expected RSC content to replace the HTML preview, got %q", detail.ContentHTML)
	}
}
