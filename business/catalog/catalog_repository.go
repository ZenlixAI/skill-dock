package catalog

import "context"

type CatalogRepository struct {
	reads  *ReadModule
	writes *WriteModule
}

type CatalogRepositoryConfig struct {
	StateStore     CatalogStateStore
	ArtifactSource ArtifactSource
}

func NewCatalogRepository(config CatalogRepositoryConfig) *CatalogRepository {
	if config.StateStore == nil {
		panic("catalog state store is required")
	}
	return &CatalogRepository{
		reads:  newReadModule(config.StateStore),
		writes: newWriteModule(config.StateStore, config.ArtifactSource),
	}
}

func (r *CatalogRepository) Ping(ctx context.Context) error {
	return r.writes.Ping(ctx)
}

func (r *CatalogRepository) ListSkills(ctx context.Context, query SkillQuery) (SkillsPage, error) {
	return r.reads.ListSkills(ctx, query)
}

func (r *CatalogRepository) ListCuratedSkills(ctx context.Context, pageSize int, cursor int) (CuratedSkillsPage, error) {
	return r.reads.ListCuratedSkills(ctx, pageSize, cursor)
}

func (r *CatalogRepository) GetSkill(ctx context.Context, packageName string) (SkillDetail, error) {
	return r.reads.GetSkill(ctx, packageName)
}

func (r *CatalogRepository) ListVersions(ctx context.Context, query VersionQuery) (VersionsPage, error) {
	return r.reads.ListVersions(ctx, query)
}

func (r *CatalogRepository) GetVersion(ctx context.Context, packageName string, version string) (SkillVersionDetail, error) {
	return r.reads.GetVersion(ctx, packageName, version)
}

func (r *CatalogRepository) ListGovernanceChanges(ctx context.Context, query GovernanceQuery) (GovernancePage, error) {
	return r.reads.ListGovernanceChanges(ctx, query)
}

func (r *CatalogRepository) ResolveInstallTarget(ctx context.Context, packageName string, version string) (InstallResolution, error) {
	return r.reads.ResolveInstallTarget(ctx, packageName, version)
}

func (r *CatalogRepository) CreateReleaseRecord(ctx context.Context, packageName string, version string, input ReleaseRecordInput) (ReleaseRecordResult, error) {
	return r.writes.CreateReleaseRecord(ctx, packageName, version, input)
}

func (r *CatalogRepository) ApplySkillGovernanceChange(ctx context.Context, packageName string, input SkillGovernanceChangeInput, actor string) (GovernanceChangeResponse, error) {
	return r.writes.ApplySkillGovernanceChange(ctx, packageName, input, actor)
}

func (r *CatalogRepository) ApplyVersionGovernanceChange(ctx context.Context, packageName string, version string, input VersionGovernanceChangeInput, actor string) (GovernanceChangeResponse, error) {
	return r.writes.ApplyVersionGovernanceChange(ctx, packageName, version, input, actor)
}

func (r *CatalogRepository) Seed(ctx context.Context, data SampleCatalogData) error {
	return r.writes.Seed(ctx, data)
}
