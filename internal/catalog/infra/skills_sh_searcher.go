package infra

import (
	"context"
	"encoding/json"
	"fmt"
	stdhtml "html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	catalog "skill_registry/internal/catalog/domain"
)

const (
	skillsShDefaultBaseURL = "https://skills.sh"
	skillsShDefaultTimeout = 8 * time.Second
	skillsShSearchMinChars = 2
	skillsShMaxSearchLimit = 100
)

var (
	skillsShSourcePattern      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*$`)
	skillsShSkillIDPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]*$`)
	skillsShCommandNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:@-]*$`)
	skillsShWeeklyInstalls     = regexp.MustCompile(`\d[\d,]*`)
)

type skillsShSearchSource struct {
	baseURL string
	client  *http.Client
}

func NewSkillsShSearchSource(baseURL string, client *http.Client) catalog.ExternalSkillSearcher {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		trimmed = skillsShDefaultBaseURL
	}
	if client == nil {
		client = &http.Client{Timeout: skillsShDefaultTimeout}
	}
	return &skillsShSearchSource{baseURL: trimmed, client: client}
}

func (s *skillsShSearchSource) SearchSkills(ctx context.Context, query catalog.ExternalSearchQuery) (catalog.ExternalSkillsPage, error) {
	text := strings.TrimSpace(query.Query)
	if len([]rune(text)) < skillsShSearchMinChars {
		return catalog.ExternalSkillsPage{Items: []catalog.ExternalSkill{}}, nil
	}

	limit := query.PageSize
	if limit <= 0 {
		limit = 20
	}
	if limit > skillsShMaxSearchLimit {
		limit = skillsShMaxSearchLimit
	}

	endpoint, err := s.endpoint("/api/search")
	if err != nil {
		return catalog.ExternalSkillsPage{}, err
	}
	params := endpoint.Query()
	params.Set("q", text)
	params.Set("limit", strconv.Itoa(limit))
	endpoint.RawQuery = params.Encode()

	var payload struct {
		Skills []struct {
			ID       string `json:"id"`
			Source   string `json:"source"`
			SkillID  string `json:"skillId"`
			Name     string `json:"name"`
			Installs int    `json:"installs"`
		} `json:"skills"`
	}
	if err := s.getJSON(ctx, endpoint.String(), &payload); err != nil {
		return catalog.ExternalSkillsPage{}, err
	}

	items := make([]catalog.ExternalSkill, 0, len(payload.Skills))
	for _, entry := range payload.Skills {
		items = append(items, s.summaryFromSearch(entry.ID, entry.Source, entry.SkillID, entry.Name, entry.Installs))
	}
	return catalog.ExternalSkillsPage{
		Items:    items,
		PageInfo: catalog.PageInfo{HasNextPage: false},
	}, nil
}

func (s *skillsShSearchSource) ListSkills(ctx context.Context, query catalog.ExternalBrowseQuery) (catalog.ExternalSkillsPage, error) {
	view := normalizeSkillsShView(query.View)
	page := query.Page
	if page < 1 {
		page = 1
	}

	endpoint, err := s.endpoint(fmt.Sprintf("/api/skills/%s/%d", url.PathEscape(view), page))
	if err != nil {
		return catalog.ExternalSkillsPage{}, err
	}

	var payload struct {
		Skills []struct {
			Source            string `json:"source"`
			SkillID           string `json:"skillId"`
			Name              string `json:"name"`
			Installs          int    `json:"installs"`
			InstallsYesterday int    `json:"installsYesterday"`
			Change            int    `json:"change"`
			WeeklyInstalls    []int  `json:"weeklyInstalls"`
			IsOfficial        bool   `json:"isOfficial"`
		} `json:"skills"`
		Total   int  `json:"total"`
		HasMore bool `json:"hasMore"`
		Page    int  `json:"page"`
	}
	if err := s.getJSON(ctx, endpoint.String(), &payload); err != nil {
		return catalog.ExternalSkillsPage{}, err
	}

	items := make([]catalog.ExternalSkill, 0, len(payload.Skills))
	for _, entry := range payload.Skills {
		item := s.summaryFromSearch("", entry.Source, entry.SkillID, entry.Name, entry.Installs)
		item.InstallsYesterday = entry.InstallsYesterday
		item.Change = entry.Change
		item.WeeklyInstalls = append([]int(nil), entry.WeeklyInstalls...)
		item.IsOfficial = entry.IsOfficial
		items = append(items, item)
	}

	hasMore := payload.HasMore || len(items) > query.PageSize
	if query.PageSize > 0 && len(items) > query.PageSize {
		items = items[:query.PageSize]
	}
	if payload.Page < 1 {
		payload.Page = page
	}

	return catalog.ExternalSkillsPage{
		Items:    items,
		PageInfo: catalog.PageInfo{HasNextPage: hasMore},
		Total:    payload.Total,
		Page:     payload.Page,
	}, nil
}

func (s *skillsShSearchSource) GetSkill(ctx context.Context, source string, skillID string) (catalog.ExternalSkillDetail, error) {
	source = strings.TrimSpace(source)
	skillID = strings.TrimSpace(skillID)
	if !skillsShSourcePattern.MatchString(source) {
		return catalog.ExternalSkillDetail{}, fmt.Errorf("skills.sh source is invalid")
	}
	if !skillsShSkillIDPattern.MatchString(skillID) {
		return catalog.ExternalSkillDetail{}, fmt.Errorf("skills.sh skill id is invalid")
	}

	detailURL := s.detailURL(source, skillID)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, detailURL, nil)
	if err != nil {
		return catalog.ExternalSkillDetail{}, err
	}
	request.Header.Set("accept", "text/html,application/xhtml+xml")
	request.Header.Set("user-agent", "skill-dock-registry/1.0")

	response, err := s.client.Do(request)
	if err != nil {
		return catalog.ExternalSkillDetail{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return catalog.ExternalSkillDetail{}, fmt.Errorf("skills.sh detail failed: %s", response.Status)
	}

	doc, err := html.Parse(response.Body)
	if err != nil {
		return catalog.ExternalSkillDetail{}, err
	}

	item := s.summaryFromSearch("", source, skillID, skillID, 0)
	item.LearnMoreURL = detailURL
	detail := catalog.ExternalSkillDetail{
		ExternalSkill: item,
		URL:           detailURL,
	}

	if title := firstElementText(doc, "title"); title != "" {
		detail.Name = strings.TrimSpace(strings.SplitN(title, " — ", 2)[0])
	}
	detail.Description = firstNonEmptyString(metaContent(doc, "name", "description"), metaContent(doc, "property", "og:description"))
	detail.DisplayName = detail.Name

	if h1s := elementTexts(doc, "h1"); len(h1s) > 0 {
		detail.Name = strings.TrimSpace(h1s[0])
		if len(h1s) > 1 {
			candidate := strings.TrimSpace(h1s[1])
			if candidate != detail.Name && len([]rune(candidate)) <= 80 && !strings.Contains(candidate, " — ") {
				detail.DisplayName = candidate
			}
		}
	}

	applySkillsShJSONLD(doc, &detail)
	detail.Topics = topicNames(doc)
	detail.WeeklyInstalls = weeklyInstallsFromDoc(doc)
	commandName := detail.Name
	if !skillsShCommandNamePattern.MatchString(commandName) {
		commandName = skillID
	}
	detail.InstallCommand = installCommandFor(source, commandName)
	previewHTML := proseHTML(doc, 1)
	if previewHTML == "" {
		previewHTML = proseHTML(doc, 0)
	}
	if fullHTML := s.fetchFullContent(ctx, detailURL); fullHTML != "" {
		detail.ContentHTML = sanitizeSkillsShHTML(fullHTML)
	} else {
		detail.ContentHTML = sanitizeSkillsShHTML(previewHTML)
	}
	detail.SummaryHTML = sanitizeSkillsShHTML(proseHTML(doc, 0))
	detail.RelatedSkills = relatedSkills(doc, s.baseURL)
	detail.IsOfficial = hasOfficialBadge(doc)

	if detail.Description == "" && detail.SummaryHTML == "" && detail.ContentHTML == "" {
		return catalog.ExternalSkillDetail{}, fmt.Errorf("skills.sh skill not found")
	}

	return detail, nil
}

func (s *skillsShSearchSource) summaryFromSearch(id string, source string, skillID string, name string, installs int) catalog.ExternalSkill {
	id = strings.TrimSpace(id)
	source = strings.TrimSpace(source)
	skillID = strings.TrimSpace(skillID)
	name = strings.TrimSpace(name)
	if id == "" && source != "" && skillID != "" {
		id = source + "/" + skillID
	}
	if name == "" {
		name = skillID
	}
	learnMoreURL := ""
	if id != "" {
		learnMoreURL = s.baseURL + "/" + id
	}
	return catalog.ExternalSkill{
		ID:                id,
		Source:            source,
		SkillID:           skillID,
		Name:              name,
		Installs:          installs,
		LearnMoreURL:      learnMoreURL,
		OriginRegistryURL: s.baseURL,
	}
}

func (s *skillsShSearchSource) endpoint(path string) (*url.URL, error) {
	return url.Parse(s.baseURL + path)
}

func (s *skillsShSearchSource) detailURL(source string, skillID string) string {
	parts := strings.Split(source, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return s.baseURL + "/" + strings.Join(parts, "/") + "/" + url.PathEscape(skillID)
}

func (s *skillsShSearchSource) getJSON(ctx context.Context, endpoint string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("accept", "application/json")
	request.Header.Set("user-agent", "skill-dock-registry/1.0")

	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("skills.sh request failed: %s", response.Status)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func normalizeSkillsShView(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "trending":
		return "trending"
	case "hot":
		return "hot"
	default:
		return "all-time"
	}
}

func applySkillsShJSONLD(doc *html.Node, detail *catalog.ExternalSkillDetail) {
	jsonLDDocuments(doc).forEach(func(payload map[string]any) {
		if valueString(payload["@type"]) != "SoftwareApplication" {
			return
		}
		detail.Description = firstNonEmptyString(valueString(payload["description"]), detail.Description)
		if publisher, ok := payload["publisher"].(map[string]any); ok {
			detail.Source = firstNonEmptyString(detail.Source, valueString(publisher["name"]))
		}
		if statistic, ok := payload["interactionStatistic"].(map[string]any); ok {
			if count, ok := statistic["userInteractionCount"]; ok {
				detail.Installs = valueInt(count)
			}
		}
	})
}

func jsonLDDocuments(doc *html.Node) stringList {
	var documents stringList
	walkHTML(doc, func(node *html.Node) {
		if node.Type != html.ElementNode || node.Data != "script" {
			return
		}
		if strings.ToLower(attribute(node, "type")) != "application/ld+json" {
			return
		}
		documents = append(documents, textContent(node))
	})
	return documents
}

type stringList []string

func (values stringList) forEach(fn func(map[string]any)) {
	for _, raw := range values {
		var payload map[string]any
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			continue
		}
		fn(payload)
	}
}

func topicNames(doc *html.Node) []string {
	seen := map[string]struct{}{}
	topics := make([]string, 0)
	stopped := false
	walkHTML(doc, func(node *html.Node) {
		if stopped {
			return
		}
		if node.Type == html.ElementNode && hasClassToken(node, "uppercase") {
			heading := headingText(node)
			if strings.EqualFold(heading, "Installation") || strings.EqualFold(heading, "Summary") {
				stopped = true
				return
			}
		}
		if node.Type != html.ElementNode || node.Data != "a" {
			return
		}
		href := attribute(node, "href")
		if !strings.HasPrefix(href, "/topic/") {
			return
		}
		name := strings.TrimSpace(textContent(node))
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		topics = append(topics, name)
	})
	return topics
}

func weeklyInstallsFromDoc(doc *html.Node) []int {
	var values []int
	walkHTML(doc, func(node *html.Node) {
		if node.Type != html.ElementNode || node.Data != "svg" {
			return
		}
		label := attribute(node, "aria-label")
		if !strings.HasPrefix(label, "Weekly installs:") {
			return
		}
		matches := skillsShWeeklyInstalls.FindAllString(label, -1)
		for _, match := range matches {
			value, err := strconv.Atoi(strings.ReplaceAll(match, ",", ""))
			if err == nil {
				values = append(values, value)
			}
		}
	})
	return values
}

func relatedSkills(doc *html.Node, baseURL string) []catalog.ExternalSkill {
	related := make([]catalog.ExternalSkill, 0)
	seen := map[string]struct{}{}
	walkHTML(doc, func(node *html.Node) {
		if node.Type != html.ElementNode || node.Data != "a" {
			return
		}
		href := attribute(node, "href")
		if !strings.HasPrefix(href, "/") || strings.HasPrefix(href, "/topic/") {
			return
		}
		parts := strings.Split(strings.Trim(href, "/"), "/")
		if len(parts) != 3 {
			return
		}
		name := ""
		if heading := firstDescendantElement(node, "h3"); heading != nil {
			name = strings.TrimSpace(textContent(heading))
		}
		if name == "" {
			return
		}
		id := strings.Join(parts, "/")
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		related = append(related, catalog.ExternalSkill{
			ID:                id,
			Source:            parts[0] + "/" + parts[1],
			SkillID:           parts[2],
			Name:              name,
			LearnMoreURL:      strings.TrimRight(baseURL, "/") + "/" + id,
			OriginRegistryURL: baseURL,
		})
	})
	if len(related) > 6 {
		related = related[:6]
	}
	return related
}

func hasOfficialBadge(doc *html.Node) bool {
	found := false
	walkHTML(doc, func(node *html.Node) {
		if found || node.Type != html.ElementNode {
			return
		}
		label := strings.ToLower(strings.TrimSpace(attribute(node, "aria-label")))
		if label == "official" || strings.Contains(label, "verified organization") {
			found = true
		}
	})
	return found
}

func installCommandFor(source string, name string) string {
	if source == "" || name == "" {
		return ""
	}
	return fmt.Sprintf("skills add %s --skill %s", source, name)
}

func proseHTML(doc *html.Node, index int) string {
	var prose []*html.Node
	walkHTML(doc, func(node *html.Node) {
		if node.Type != html.ElementNode {
			return
		}
		if hasClassToken(node, "prose") {
			prose = append(prose, node)
		}
	})
	if index < 0 || index >= len(prose) {
		return ""
	}
	var builder strings.Builder
	for child := prose[index].FirstChild; child != nil; child = child.NextSibling {
		_ = html.Render(&builder, child)
	}
	return builder.String()
}

func sanitizeSkillsShHTML(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	nodes, err := html.ParseFragment(strings.NewReader(raw), &html.Node{
		Type:     html.ElementNode,
		Data:     "div",
		DataAtom: atom.Div,
	})
	if err != nil {
		return ""
	}

	var builder strings.Builder
	for _, node := range nodes {
		sanitizeNode(&builder, node)
	}
	return strings.TrimSpace(builder.String())
}

func sanitizeNode(builder *strings.Builder, node *html.Node) {
	if node == nil {
		return
	}
	switch node.Type {
	case html.TextNode:
		builder.WriteString(stdhtml.EscapeString(node.Data))
		return
	case html.DocumentNode:
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			sanitizeNode(builder, child)
		}
		return
	case html.ElementNode:
		// fall through
	default:
		return
	}

	tag := strings.ToLower(node.Data)
	if tag == "script" || tag == "style" || tag == "iframe" || tag == "object" || tag == "embed" || tag == "form" || tag == "input" || tag == "button" {
		return
	}
	if !allowedSkillsShTag(tag) {
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			sanitizeNode(builder, child)
		}
		return
	}

	builder.WriteString("<")
	builder.WriteString(tag)
	if tag == "a" {
		href := sanitizeSkillsShHref(attribute(node, "href"))
		if href != "" {
			builder.WriteString(` href="`)
			builder.WriteString(stdhtml.EscapeString(href))
			builder.WriteString(`" rel="noreferrer noopener" target="_blank"`)
		}
	}
	if tag == "th" || tag == "td" {
		if colspan := numericAttribute(node, "colspan"); colspan != "" {
			builder.WriteString(` colspan="`)
			builder.WriteString(colspan)
			builder.WriteString(`"`)
		}
		if rowspan := numericAttribute(node, "rowspan"); rowspan != "" {
			builder.WriteString(` rowspan="`)
			builder.WriteString(rowspan)
			builder.WriteString(`"`)
		}
	}
	builder.WriteString(">")

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		sanitizeNode(builder, child)
	}
	builder.WriteString("</")
	builder.WriteString(tag)
	builder.WriteString(">")
}

func allowedSkillsShTag(tag string) bool {
	switch tag {
	case "a", "p", "br", "hr", "strong", "b", "em", "i", "code", "pre", "ul", "ol", "li", "blockquote",
		"h1", "h2", "h3", "h4", "h5", "h6", "table", "thead", "tbody", "tr", "th", "td", "span", "div", "del", "sup", "sub":
		return true
	default:
		return false
	}
}

func sanitizeSkillsShHref(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "#") {
		return ""
	}
	if strings.HasPrefix(raw, "/") {
		return "https://www.skills.sh" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ""
	}
	return raw
}

func numericAttribute(node *html.Node, name string) string {
	value := strings.TrimSpace(attribute(node, name))
	if value == "" {
		return ""
	}
	if _, err := strconv.Atoi(value); err != nil {
		return ""
	}
	return value
}

func metaContent(doc *html.Node, attributeName string, attributeValue string) string {
	content := ""
	walkHTML(doc, func(node *html.Node) {
		if content != "" || node.Type != html.ElementNode || node.Data != "meta" {
			return
		}
		if strings.EqualFold(attribute(node, attributeName), attributeValue) {
			content = strings.TrimSpace(attribute(node, "content"))
		}
	})
	return content
}

func firstElementText(doc *html.Node, tag string) string {
	texts := elementTexts(doc, tag)
	if len(texts) == 0 {
		return ""
	}
	return texts[0]
}

func elementTexts(doc *html.Node, tag string) []string {
	var texts []string
	walkHTML(doc, func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == tag {
			texts = append(texts, strings.TrimSpace(textContent(node)))
		}
	})
	return texts
}

func firstChildElement(node *html.Node, tag string) *html.Node {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == tag {
			return child
		}
	}
	return nil
}

func firstDescendantElement(node *html.Node, tag string) *html.Node {
	var found *html.Node
	walkHTML(node, func(current *html.Node) {
		if found != nil || current == node {
			return
		}
		if current.Type == html.ElementNode && current.Data == tag {
			found = current
		}
	})
	return found
}

func hasClassToken(node *html.Node, token string) bool {
	for _, class := range strings.Fields(attribute(node, "class")) {
		if class == token {
			return true
		}
	}
	return false
}

func attribute(node *html.Node, name string) string {
	if node == nil {
		return ""
	}
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, name) {
			return attr.Val
		}
	}
	return ""
}

func textContent(node *html.Node) string {
	var builder strings.Builder
	walkHTML(node, func(current *html.Node) {
		if current.Type == html.TextNode {
			builder.WriteString(current.Data)
		}
	})
	return builder.String()
}

func directTextContent(node *html.Node) string {
	var builder strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.TextNode {
			builder.WriteString(child.Data)
		}
	}
	return builder.String()
}

func headingText(node *html.Node) string {
	if text := strings.TrimSpace(directTextContent(node)); text != "" {
		return text
	}
	if span := firstChildElement(node, "span"); span != nil {
		return strings.TrimSpace(textContent(span))
	}
	return ""
}

func walkHTML(node *html.Node, fn func(*html.Node)) {
	if node == nil {
		return
	}
	fn(node)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		walkHTML(child, fn)
	}
}

func (s *skillsShSearchSource) fetchFullContent(ctx context.Context, detailURL string) string {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, detailURL, nil)
	if err != nil {
		return ""
	}
	request.Header.Set("RSC", "1")
	request.Header.Set("accept", "text/x-component")
	request.Header.Set("user-agent", "skill-dock-registry/1.0")

	response, err := s.client.Do(request)
	if err != nil {
		return ""
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return ""
	}

	preview := extractRSCJSONString(body, "previewHtml")
	if preview == "" {
		return ""
	}
	rest := extractRSCJSONString(body, "restHtml")
	if strings.HasPrefix(rest, "$") {
		rest = extractRSCChunk(body, strings.TrimPrefix(rest, "$"))
	}
	return preview + rest
}

func extractRSCJSONString(body []byte, key string) string {
	marker := []byte(`"` + key + `":"`)
	index := strings.Index(string(body), string(marker))
	if index < 0 {
		return ""
	}
	start := index + len(marker)
	var builder strings.Builder
	escaped := false
	for i := start; i < len(body); i++ {
		character := body[i]
		if escaped {
			builder.WriteByte('\\')
			builder.WriteByte(character)
			escaped = false
			continue
		}
		if character == '\\' {
			escaped = true
			continue
		}
		if character == '"' {
			var decoded string
			if err := json.Unmarshal([]byte(`"`+builder.String()+`"`), &decoded); err != nil {
				return ""
			}
			return decoded
		}
		builder.WriteByte(character)
	}
	return ""
}

func extractRSCChunk(body []byte, id string) string {
	marker := []byte(id + ":T")
	index := strings.Index(string(body), string(marker))
	if index < 0 {
		return ""
	}
	start := index + len(marker)
	comma := strings.IndexByte(string(body[start:]), ',')
	if comma < 0 {
		return ""
	}
	contentStart := start + comma + 1
	content := string(body[contentStart:])

	// The rest of the SKILL.md is followed by the RSC row that renders the
	// preview/summary component. RSC rows are not newline-delimited inside a
	// text chunk, so cut at that row rather than at the next newline.
	if previewIndex := strings.Index(content, `"previewHtml":"`); previewIndex > 0 {
		beforePreview := content[:previewIndex]
		rows := regexp.MustCompile(`[0-9a-f]{1,3}:\["\$",`).FindAllStringIndex(beforePreview, -1)
		if len(rows) > 0 {
			return beforePreview[:rows[len(rows)-1][0]]
		}
	}

	next := regexp.MustCompile(`\n[0-9a-f]{1,3}:`).FindStringIndex(content)
	if next != nil {
		content = content[:next[0]]
	}
	return content
}

func valueString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func valueInt(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	default:
		return 0
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
