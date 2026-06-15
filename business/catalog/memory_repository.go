package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"

	internaldomain "skill_registry/internal/catalog/domain"
	internalinfra "skill_registry/internal/catalog/infra"
)

type MemoryRepositoryOptions struct {
	ArtifactSource ArtifactSource
	AggregateStore AggregateStore
}

type skillRecord = internaldomain.SkillRecord
type SampleCatalogData = internaldomain.SampleCatalogData

func NewMemoryRepository(data SampleCatalogData, options ...MemoryRepositoryOptions) *CatalogRepository {
	config := MemoryRepositoryOptions{}
	if len(options) > 0 {
		config = options[0]
	}

	artifactSource := config.ArtifactSource
	if artifactSource == nil {
		artifactSource = NewStaticArtifactSource(data.Artifacts)
	}

	return NewCatalogRepository(CatalogRepositoryConfig{
		StateStore: NewMemoryAggregateStateStore(data, MemoryAggregateStateStoreOptions{
			AggregateStore: config.AggregateStore,
		}),
		ArtifactSource: artifactSource,
	})
}

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
	return internalinfra.SampleData()
}

func SampleDataWithRegistryBase(registryBase string) SampleCatalogData {
	return internalinfra.SampleDataWithRegistryBase(registryBase)
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

func releaseArtifactKey(packageName string, version string) string {
	return packageName + "@" + version
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
