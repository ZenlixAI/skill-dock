package catalog

import (
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
	aggregates := internalinfra.AggregatesFromSampleData(data)
	result := make([]AgentSkillAggregate, 0, len(aggregates))
	for _, aggregate := range aggregates {
		result = append(result, fromInternalAggregate(aggregate))
	}
	return result
}

func latestReplacementRecommendation(packageName string, version string, records []GovernanceRecord) *string {
	return internalinfra.LatestReplacementRecommendation(packageName, version, records)
}

func refreshLatestVersion(record *skillRecord) {
	internalinfra.RefreshLatestVersion(record)
}

func matchesSkillQuery(skill SkillDetail, query string) bool {
	return internalinfra.MatchesSkillQuery(skill, query)
}

func sortSkills(items []SkillSummary, sortKey string, query string) {
	internalinfra.SortSkills(items, sortKey, query)
}

func skillRelevanceScore(item SkillSummary, query string) int {
	return internalinfra.SkillRelevanceScore(item, query)
}

func compareExactVersions(left string, right string) int {
	return internalinfra.CompareExactVersions(left, right)
}

func paginate[T any](items []T, cursor int, pageSize int) ([]T, PageInfo, error) {
	return internalinfra.Paginate(items, cursor, pageSize)
}

func formatCursorToken(offset int) string {
	return internalinfra.FormatCursorToken(offset)
}

func SampleData() SampleCatalogData {
	return internalinfra.SampleData()
}

func SampleDataWithRegistryBase(registryBase string) SampleCatalogData {
	return internalinfra.SampleDataWithRegistryBase(registryBase)
}

func newVersion(registryBase string, packageName string, version string, versionStatus VersionStatus, policy InstallPolicy, publishedAt time.Time) SkillVersionDetail {
	return internalinfra.NewVersion(registryBase, packageName, version, versionStatus, policy, publishedAt)
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
	return internalinfra.NewMaterializedVersion(registryBase, packageName, version, versionStatus, policy, publishedAt, displayName, description, components)
}

func stringPtr(value string) *string {
	return internalinfra.StringPtr(value)
}

func defaultActor(actor string) string {
	return internalinfra.DefaultActor(actor)
}

func validateReleaseRecordInput(input ReleaseRecordInput, artifact ReleaseArtifact) *Error {
	return internalinfra.ValidateReleaseRecordInput(input, artifact)
}

func sameStringSet(left []string, right []string) bool {
	return internalinfra.SameStringSet(left, right)
}

func releaseArtifactKey(packageName string, version string) string {
	return packageName + "@" + version
}
