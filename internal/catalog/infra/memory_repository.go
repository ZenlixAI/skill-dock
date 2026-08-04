package infra

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"

	. "skill_registry/internal/catalog/domain"
	internaldomain "skill_registry/internal/catalog/domain"
)

type skillRecord = internaldomain.SkillRecord

func aggregatesFromSampleData(data SampleCatalogData) []AgentSkillAggregate {
	aggregates := make([]AgentSkillAggregate, 0, len(data.Skills))
	governanceByPackage := make(map[string][]GovernanceRecord)
	for _, record := range data.Governance {
		governanceByPackage[record.PackageName] = append(governanceByPackage[record.PackageName], record)
	}
	for _, skill := range data.Skills {
		aggregates = append(aggregates, AgentSkillAggregate{
			Revision: 1,
			Skill: SkillSnapshot{
				Detail:         skill.Detail,
				CurationReason: skill.CurationReason,
				Versions:       append([]SkillVersionDetail(nil), skill.Versions...),
			},
			Governance: append([]GovernanceRecord(nil), governanceByPackage[skill.Detail.PackageName]...),
		})
	}
	sort.Slice(aggregates, func(i int, j int) bool {
		return aggregates[i].Skill.Detail.PackageName < aggregates[j].Skill.Detail.PackageName
	})
	return aggregates
}

func latestReplacementRecommendation(packageName string, version string, records []GovernanceRecord) *string {
	for i := len(records) - 1; i >= 0; i-- {
		record := records[i]
		if record.PackageName != packageName {
			continue
		}
		if record.Version == nil || *record.Version != version {
			continue
		}
		if record.ReplacementRecommendation != nil {
			return record.ReplacementRecommendation
		}
	}
	return nil
}

func refreshLatestVersion(record *skillRecord) {
	var latest *SkillVersionDetail
	for _, version := range record.Versions {
		if version.VersionStatus == VersionStatusYanked {
			continue
		}
		if latest == nil || compareExactVersions(version.Version, latest.Version) > 0 {
			candidate := version
			latest = &candidate
		}
	}
	if latest == nil {
		for _, version := range record.Versions {
			if latest == nil || compareExactVersions(version.Version, latest.Version) > 0 {
				candidate := version
				latest = &candidate
			}
		}
	}
	if latest == nil {
		record.Detail.LatestVersion = nil
		record.Detail.LatestVersionStatus = nil
		return
	}

	record.Detail.LatestVersion = stringPtr(latest.Version)
	record.Detail.LatestVersionStatus = stringPtr(string(latest.VersionStatus))
}

func matchesSkillQuery(skill SkillDetail, query string) bool {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return true
	}
	return strings.Contains(strings.ToLower(skill.PackageName), needle) ||
		strings.Contains(strings.ToLower(skill.DisplayName), needle) ||
		strings.Contains(strings.ToLower(skill.Description), needle)
}

func sortSkills(items []SkillSummary, sortKey string, query string) {
	sort.Slice(items, func(i int, j int) bool {
		switch sortKey {
		case "package_name_asc":
			return items[i].PackageName < items[j].PackageName
		case "updated_at_desc":
			if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
				return items[i].UpdatedAt.After(items[j].UpdatedAt)
			}
			return items[i].PackageName < items[j].PackageName
		default:
			scoreI := skillRelevanceScore(items[i], query)
			scoreJ := skillRelevanceScore(items[j], query)
			if scoreI == scoreJ {
				if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
					return items[i].UpdatedAt.After(items[j].UpdatedAt)
				}
				return items[i].PackageName < items[j].PackageName
			}
			return scoreI > scoreJ
		}
	})
}

func skillRelevanceScore(item SkillSummary, query string) int {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return 0
	}
	score := 0
	if strings.Contains(strings.ToLower(item.PackageName), needle) {
		score += 3
	}
	if strings.Contains(strings.ToLower(item.DisplayName), needle) {
		score += 2
	}
	if strings.Contains(strings.ToLower(item.Description), needle) {
		score++
	}
	return score
}

func compareExactVersions(left string, right string) int {
	lMajor, lMinor, lPatch := parseVersionCore(left)
	rMajor, rMinor, rPatch := parseVersionCore(right)

	if lMajor != rMajor {
		return compareInts(lMajor, rMajor)
	}
	if lMinor != rMinor {
		return compareInts(lMinor, rMinor)
	}
	return compareInts(lPatch, rPatch)
}

func parseVersionCore(version string) (int, int, int) {
	core := version
	if idx := strings.Index(core, "-"); idx >= 0 {
		core = core[:idx]
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return 0, 0, 0
	}

	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])
	return major, minor, patch
}

func compareInts(left int, right int) int {
	switch {
	case left > right:
		return 1
	case left < right:
		return -1
	default:
		return 0
	}
}

func isExactVersion(version string) bool {
	parts := strings.Split(strings.TrimSpace(version), ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}
	return true
}

func paginate[T any](items []T, cursor int, pageSize int) ([]T, PageInfo, error) {
	if cursor < 0 {
		return nil, PageInfo{}, NewError(400, "invalid_request", "cursor must be greater than or equal to 0.", map[string]any{"cursor": cursor})
	}
	if cursor > len(items) {
		return nil, PageInfo{}, NewError(400, "invalid_request", "cursor is out of range.", map[string]any{"cursor": cursor})
	}

	end := cursor + pageSize
	if end > len(items) {
		end = len(items)
	}

	var nextCursor *string
	hasNextPage := end < len(items)
	if hasNextPage {
		cursorValue := formatCursorToken(end)
		nextCursor = &cursorValue
	}

	return append([]T(nil), items[cursor:end]...), PageInfo{
		NextCursor:  nextCursor,
		HasNextPage: hasNextPage,
	}, nil
}

func formatCursorToken(offset int) string {
	return "cursor_" + strconv.Itoa(offset)
}

func SampleData() SampleCatalogData {
	return SampleDataWithRegistryBase("https://npm.internal.example.com")
}

func SampleDataWithRegistryBase(registryBase string) SampleCatalogData {
	latestPublished := "1.2.0"
	latestPublishedStatus := "published"
	curatedVersion := "0.9.0"
	curatedVersionStatus := "published"
	retiredVersion := "2.0.0"
	retiredVersionStatus := "published"

	pdfVersions := []SkillVersionDetail{
		newMaterializedVersion(registryBase, "@platform.skill/pdf-skill", "1.0.0", VersionStatusPublished, InstallPolicyResolvable, time.Date(2026, 5, 20, 8, 0, 0, 0, time.UTC), "PDF Skill", "PDF processing workflow", []string{"commands", "scripts"}),
		newMaterializedVersion(registryBase, "@platform.skill/pdf-skill", "1.1.0", VersionStatusDeprecated, InstallPolicyResolvable, time.Date(2026, 5, 23, 8, 0, 0, 0, time.UTC), "PDF Skill", "PDF processing workflow", []string{"commands", "scripts"}),
		newMaterializedVersion(registryBase, "@platform.skill/pdf-skill", "1.2.0", VersionStatusPublished, InstallPolicyResolvable, time.Date(2026, 5, 28, 8, 20, 0, 0, time.UTC), "PDF Skill", "PDF processing workflow", []string{"commands", "scripts"}),
		newMaterializedVersion(registryBase, "@platform.skill/pdf-skill", "1.3.0", VersionStatusYanked, InstallPolicyBlocked, time.Date(2026, 5, 29, 8, 20, 0, 0, time.UTC), "PDF Skill", "PDF processing workflow", []string{"commands", "scripts"}),
	}

	curatedVersions := []SkillVersionDetail{
		newMaterializedVersion(registryBase, "@platform.skill/code-review", "0.9.0", VersionStatusPublished, InstallPolicyResolvable, time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC), "Code Review Accelerator", "Structured code review prompts and checklists", []string{"commands", "scripts"}),
	}

	retiredVersions := []SkillVersionDetail{
		newMaterializedVersion(registryBase, "@community/legacy-notes", "2.0.0", VersionStatusPublished, InstallPolicyResolvable, time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC), "Legacy Notes", "Old note-taking workflow retained for migration", []string{"commands", "scripts"}),
	}

	yankedVersion := "1.3.0"
	replacementRecommendation := "1.2.0"

	return SampleCatalogData{
		Skills: []skillRecord{
			{
				Detail: SkillDetail{
					SkillSummary: SkillSummary{
						PackageName:         "@platform.skill/pdf-skill",
						DisplayName:         "PDF Skill",
						Description:         "PDF processing workflow",
						SourceType:          SourceTypeImported,
						Status:              SkillStatusActive,
						LatestVersion:       &latestPublished,
						LatestVersionStatus: &latestPublishedStatus,
						UpdatedAt:           time.Date(2026, 5, 29, 9, 0, 0, 0, time.UTC),
					},
					CreatedAt: time.Date(2026, 5, 20, 2, 0, 0, 0, time.UTC),
				},
				Versions: pdfVersions,
			},
			{
				Detail: SkillDetail{
					SkillSummary: SkillSummary{
						PackageName:         "@platform.skill/code-review",
						DisplayName:         "Code Review Accelerator",
						Description:         "Structured code review prompts and checklists",
						SourceType:          SourceTypeNative,
						Status:              SkillStatusActive,
						LatestVersion:       &curatedVersion,
						LatestVersionStatus: &curatedVersionStatus,
						UpdatedAt:           time.Date(2026, 5, 27, 7, 0, 0, 0, time.UTC),
					},
					CreatedAt: time.Date(2026, 5, 19, 3, 0, 0, 0, time.UTC),
				},
				CurationReason: "featured",
				Versions:       curatedVersions,
			},
			{
				Detail: SkillDetail{
					SkillSummary: SkillSummary{
						PackageName:         "@community/legacy-notes",
						DisplayName:         "Legacy Notes",
						Description:         "Old note-taking workflow retained for migration",
						SourceType:          SourceTypeImported,
						Status:              SkillStatusRetired,
						LatestVersion:       &retiredVersion,
						LatestVersionStatus: &retiredVersionStatus,
						UpdatedAt:           time.Date(2026, 5, 25, 6, 0, 0, 0, time.UTC),
					},
					CreatedAt: time.Date(2026, 5, 10, 1, 0, 0, 0, time.UTC),
				},
				Versions: retiredVersions,
			},
		},
		Governance: []GovernanceRecord{
			{
				ID:                        "gov_01hxyank",
				TargetType:                GovernanceTargetVersion,
				PackageName:               "@platform.skill/pdf-skill",
				Version:                   &yankedVersion,
				Action:                    GovernanceActionYank,
				Reason:                    "security issue",
				ReplacementRecommendation: &replacementRecommendation,
				ChangedBy:                 "user_123",
				ChangedAt:                 time.Date(2026, 5, 29, 9, 10, 0, 0, time.UTC),
			},
			{
				ID:          "gov_01hxretire",
				TargetType:  GovernanceTargetSkill,
				PackageName: "@community/legacy-notes",
				Action:      GovernanceActionRetire,
				Reason:      "merged into successor skill",
				ChangedBy:   "user_456",
				ChangedAt:   time.Date(2026, 5, 26, 9, 10, 0, 0, time.UTC),
			},
		},
		Artifacts: []ReleaseArtifact{
			{
				SkillDisplayName: "Agent Notes",
				SkillDescription: "Capture agent workflow notes",
				SkillSourceType:  SourceTypePlatformCreated,
				Version:          newMaterializedVersion(registryBase, "@platform.skill/agent-notes", "0.1.0", VersionStatusPublished, InstallPolicyResolvable, time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC), "Agent Notes", "Capture agent workflow notes", []string{"commands", "scripts"}),
			},
			{
				SkillDisplayName: "PDF Skill",
				SkillDescription: "PDF processing workflow",
				SkillSourceType:  SourceTypeImported,
				Version:          newMaterializedVersion(registryBase, "@platform.skill/pdf-skill", "1.2.0", VersionStatusPublished, InstallPolicyResolvable, time.Date(2026, 5, 28, 8, 20, 0, 0, time.UTC), "PDF Skill", "PDF processing workflow", []string{"commands", "scripts"}),
			},
			{
				SkillDisplayName: "PDF Skill",
				SkillDescription: "PDF processing workflow",
				SkillSourceType:  SourceTypeImported,
				Version: func() SkillVersionDetail {
					version := newMaterializedVersion(registryBase, "@platform.skill/pdf-skill", "1.1.0", VersionStatusDeprecated, InstallPolicyResolvable, time.Date(2026, 5, 23, 8, 0, 0, 0, time.UTC), "PDF Skill", "PDF processing workflow", []string{"commands", "scripts"})
					version.Bundle.SkillManifestDigest = "sha256:conflict"
					return version
				}(),
			},
			{
				SkillDisplayName: "Broken Bundle",
				SkillDescription: "Invalid bundle for testing",
				SkillSourceType:  SourceTypeImported,
				Version:          newMaterializedVersion(registryBase, "@platform.skill/broken-bundle", "0.1.0", VersionStatusPublished, InstallPolicyResolvable, time.Date(2026, 5, 30, 11, 0, 0, 0, time.UTC), "Broken Bundle", "Invalid bundle for testing", []string{"commands", "scripts"}),
				ValidationError:  "single-skill bundle must contain exactly one root SKILL.md",
			},
			{
				SkillDisplayName: "Registry Down",
				SkillDescription: "Simulate an upstream outage",
				SkillSourceType:  SourceTypeImported,
				Version:          newMaterializedVersion(registryBase, "@platform.skill/registry-down", "0.1.0", VersionStatusPublished, InstallPolicyResolvable, time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC), "Registry Down", "Simulate an upstream outage", []string{"commands", "scripts"}),
				RegistryDown:     true,
			},
		},
	}
}

func newVersion(registryBase string, packageName string, version string, versionStatus VersionStatus, policy InstallPolicy, publishedAt time.Time) SkillVersionDetail {
	return SkillVersionDetail{
		SkillVersionSummary: SkillVersionSummary{
			PackageName:   packageName,
			Version:       version,
			VersionStatus: versionStatus,
			InstallPolicy: policy,
			PublishedAt:   publishedAt,
		},
		Registry: RegistryMetadata{
			NPMRegistry: strings.TrimRight(registryBase, "/"),
			TarballURL:  strings.TrimRight(registryBase, "/") + "/" + strings.TrimPrefix(packageName, "@") + "/-/" + packageBasename(packageName) + "-" + version + ".tgz",
			Integrity:   "sha512-" + strings.ReplaceAll(version, ".", ""),
			Shasum:      "sha1-" + strings.ReplaceAll(version, ".", ""),
			PublishedAt: publishedAt,
			DistTags:    []string{"latest"},
		},
		Bundle: BundleMetadata{
			SkillManifestDigest: "sha256:" + strings.ReplaceAll(version, ".", ""),
			Components:          []string{"commands", "scripts"},
		},
	}
}

func newMaterializedVersion(
	registryBase string,
	packageName string,
	version string,
	versionStatus VersionStatus,
	policy InstallPolicy,
	publishedAt time.Time,
	displayName string,
	description string,
	components []string,
) SkillVersionDetail {
	detail := newVersion(registryBase, packageName, version, versionStatus, policy, publishedAt)
	detail.Bundle.Components = append([]string(nil), components...)
	artifact := ReleaseArtifact{
		SkillDisplayName: displayName,
		SkillDescription: description,
		Version:          detail,
	}
	tarball := buildMockTarball(artifact)
	integrity, shasum := digestMockTarball(tarball)
	detail.Registry.Integrity = integrity
	detail.Registry.Shasum = shasum
	skillMD := buildMockSkillMD(artifact)
	sum := sha256.Sum256([]byte(skillMD))
	detail.Bundle.SkillManifestDigest = "sha256:" + hex.EncodeToString(sum[:])
	return detail
}

func packageBasename(packageName string) string {
	parts := strings.Split(packageName, "/")
	return parts[len(parts)-1]
}

func stringPtr(value string) *string {
	return &value
}

func defaultActor(actor string) string {
	if strings.TrimSpace(actor) == "" {
		return "system"
	}
	return actor
}

func validateReleaseRecordInput(input ReleaseRecordInput, artifact ReleaseArtifact) *Error {
	if input.PublishRegistry != nil && strings.TrimSpace(*input.PublishRegistry) != artifact.Version.Registry.NPMRegistry {
		return NewError(422, "invalid_request", "publishRegistry does not match the resolved registry metadata.", map[string]any{
			"publishRegistry": *input.PublishRegistry,
			"npmRegistry":     artifact.Version.Registry.NPMRegistry,
		})
	}
	if input.ManifestDigest != nil && strings.TrimSpace(*input.ManifestDigest) != artifact.Version.Bundle.SkillManifestDigest {
		return NewError(422, "invalid_request", "manifestDigest does not match the resolved bundle metadata.", map[string]any{
			"manifestDigest":      *input.ManifestDigest,
			"skillManifestDigest": artifact.Version.Bundle.SkillManifestDigest,
		})
	}
	if len(input.Components) > 0 && !sameStringSet(input.Components, artifact.Version.Bundle.Components) {
		return NewError(422, "invalid_request", "components do not match the resolved bundle metadata.", map[string]any{
			"components":         input.Components,
			"resolvedComponents": artifact.Version.Bundle.Components,
		})
	}
	return nil
}

func sameStringSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	normalize := func(values []string) []string {
		result := append([]string(nil), values...)
		for i := range result {
			result[i] = strings.TrimSpace(result[i])
		}
		sort.Strings(result)
		return result
	}
	leftNormalized := normalize(left)
	rightNormalized := normalize(right)
	for i := range leftNormalized {
		if leftNormalized[i] != rightNormalized[i] {
			return false
		}
	}
	return true
}

func buildMockTarball(artifact ReleaseArtifact) []byte {
	files := map[string]string{
		"package/SKILL.md":     buildMockSkillMD(artifact),
		"package/package.json": buildMockPackageJSON(artifact),
	}

	for _, component := range artifact.Version.Bundle.Components {
		switch component {
		case "mcp":
			files["package/.mcp.json"] = "{}"
		case "plugin":
			files["package/plugin.json"] = `{"name":"compat"}`
		default:
			files["package/"+component+"/placeholder.txt"] = "placeholder"
		}
	}

	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		content := files[name]
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		_ = tarWriter.WriteHeader(header)
		_, _ = tarWriter.Write([]byte(content))
	}
	_ = tarWriter.Close()
	_ = gzipWriter.Close()
	return archive.Bytes()
}

func digestMockTarball(tarball []byte) (string, string) {
	sha512Sum := sha512.Sum512(tarball)
	sha1Sum := sha1.Sum(tarball)
	return "sha512-" + base64.StdEncoding.EncodeToString(sha512Sum[:]), hex.EncodeToString(sha1Sum[:])
}

func buildMockSkillMD(artifact ReleaseArtifact) string {
	return "---\nname: " + artifact.SkillDisplayName + "\ndescription: " + artifact.SkillDescription + "\n---\n"
}

func buildMockPackageJSON(artifact ReleaseArtifact) string {
	return `{"name":"` + artifact.Version.PackageName + `","version":"` + artifact.Version.Version + `"}`
}

func AggregatesFromSampleData(data SampleCatalogData) []AgentSkillAggregate {
	return aggregatesFromSampleData(data)
}

func MatchesSkillQuery(skill SkillDetail, query string) bool {
	return matchesSkillQuery(skill, query)
}

func SortSkills(items []SkillSummary, sortKey string, query string) {
	sortSkills(items, sortKey, query)
}

func CompareExactVersions(left string, right string) int {
	return compareExactVersions(left, right)
}

func Paginate[T any](items []T, cursor int, pageSize int) ([]T, PageInfo, error) {
	return paginate(items, cursor, pageSize)
}
