package infra

import (
	"context"
	"database/sql"

	catalogapp "skill_registry/internal/catalog/application"
	catalogdomain "skill_registry/internal/catalog/domain"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type CatalogRepository struct {
	reads  *catalogapp.ReadService
	writes *catalogapp.WriteService
}

type RepositoryConfig struct {
	StateStore     catalogdomain.CatalogStateStore
	ArtifactSource catalogdomain.ArtifactSource
}

type MemoryRepositoryOptions struct {
	ArtifactSource catalogdomain.ArtifactSource
	AggregateStore catalogdomain.AggregateStore
}

type SQLRepositoryOptions struct {
	ArtifactSource catalogdomain.ArtifactSource
	DisableSeed    bool
}

func NewRepository(config RepositoryConfig) *CatalogRepository {
	if config.StateStore == nil {
		panic("catalog state store is required")
	}
	return &CatalogRepository{
		reads:  catalogapp.NewReadService(config.StateStore),
		writes: catalogapp.NewWriteService(config.StateStore, config.ArtifactSource),
	}
}

func NewMemoryRepository(data catalogdomain.SampleCatalogData, options ...MemoryRepositoryOptions) *CatalogRepository {
	config := MemoryRepositoryOptions{}
	if len(options) > 0 {
		config = options[0]
	}

	artifactSource := config.ArtifactSource
	if artifactSource == nil {
		artifactSource = NewStaticArtifactSource(data.Artifacts)
	}

	return NewRepository(RepositoryConfig{
		StateStore: NewMemoryAggregateStateStore(data, MemoryAggregateStateStoreOptions{
			AggregateStore: config.AggregateStore,
		}),
		ArtifactSource: artifactSource,
	})
}

func NewSQLRepository(ctx context.Context, databaseURL string, seedData catalogdomain.SampleCatalogData, options SQLRepositoryOptions) (*CatalogRepository, error) {
	db, _, store, err := OpenSQLStateStore(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return newSQLRepositoryWithStore(ctx, store, seedData, options, db)
}

func NewSQLRepositoryWithDB(ctx context.Context, db *sql.DB, driverName string, seedData catalogdomain.SampleCatalogData, options SQLRepositoryOptions) (*CatalogRepository, error) {
	store, err := PrepareSQLStateStore(ctx, db, driverName)
	if err != nil {
		return nil, err
	}
	return newSQLRepositoryWithStore(ctx, store, seedData, options, nil)
}

func newSQLRepositoryWithStore(ctx context.Context, store *SQLStateStore, seedData catalogdomain.SampleCatalogData, options SQLRepositoryOptions, ownedDB *sql.DB) (*CatalogRepository, error) {
	artifactSource := options.ArtifactSource
	if artifactSource == nil {
		artifactSource = NewStaticArtifactSource(seedData.Artifacts)
	}

	repository := NewRepository(RepositoryConfig{
		StateStore:     store,
		ArtifactSource: artifactSource,
	})
	if !options.DisableSeed {
		if err := repository.Seed(ctx, seedData); err != nil {
			if ownedDB != nil {
				_ = ownedDB.Close()
			}
			return nil, err
		}
	}
	return repository, nil
}

func (r *CatalogRepository) Ping(ctx context.Context) error {
	return r.writes.Ping(ctx)
}

func (r *CatalogRepository) ListSkills(ctx context.Context, query catalogdomain.SkillQuery) (catalogdomain.SkillsPage, error) {
	return r.reads.ListSkills(ctx, query)
}

func (r *CatalogRepository) ListCuratedSkills(ctx context.Context, query catalogdomain.CuratedSkillQuery) (catalogdomain.CuratedSkillsPage, error) {
	return r.reads.ListCuratedSkills(ctx, query)
}

func (r *CatalogRepository) GetSkill(ctx context.Context, packageName string) (catalogdomain.SkillDetail, error) {
	return r.reads.GetSkill(ctx, packageName)
}

func (r *CatalogRepository) ListVersions(ctx context.Context, query catalogdomain.VersionQuery) (catalogdomain.VersionsPage, error) {
	return r.reads.ListVersions(ctx, query)
}

func (r *CatalogRepository) GetVersion(ctx context.Context, packageName string, version string) (catalogdomain.SkillVersionDetail, error) {
	return r.reads.GetVersion(ctx, packageName, version)
}

func (r *CatalogRepository) ListGovernanceChanges(ctx context.Context, query catalogdomain.GovernanceQuery) (catalogdomain.GovernancePage, error) {
	return r.reads.ListGovernanceChanges(ctx, query)
}

func (r *CatalogRepository) ResolveInstallTarget(ctx context.Context, packageName string, version string) (catalogdomain.InstallResolution, error) {
	return r.reads.ResolveInstallTarget(ctx, packageName, version)
}

func (r *CatalogRepository) CreateReleaseRecord(ctx context.Context, packageName string, version string, input catalogdomain.ReleaseRecordInput) (catalogdomain.ReleaseRecordResult, error) {
	return r.writes.CreateReleaseRecord(ctx, packageName, version, input)
}

func (r *CatalogRepository) ApplySkillGovernanceChange(ctx context.Context, packageName string, input catalogdomain.SkillGovernanceChangeInput, actor string) (catalogdomain.GovernanceChangeResponse, error) {
	return r.writes.ApplySkillGovernanceChange(ctx, packageName, input, actor)
}

func (r *CatalogRepository) ApplyVersionGovernanceChange(ctx context.Context, packageName string, version string, input catalogdomain.VersionGovernanceChangeInput, actor string) (catalogdomain.GovernanceChangeResponse, error) {
	return r.writes.ApplyVersionGovernanceChange(ctx, packageName, version, input, actor)
}

func (r *CatalogRepository) Seed(ctx context.Context, data catalogdomain.SampleCatalogData) error {
	return r.writes.Seed(ctx, data)
}

func (r *CatalogRepository) CatalogReadService() *catalogapp.ReadService {
	return r.reads
}

func (r *CatalogRepository) CatalogWriteService() *catalogapp.WriteService {
	return r.writes
}
