package catalog

import (
	"context"
	"database/sql"

	internalinfra "skill_registry/internal/catalog/infra"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type SQLRepositoryOptions struct {
	ArtifactSource ArtifactSource
	DisableSeed    bool
}

func NewSQLRepository(ctx context.Context, databaseURL string, seedData SampleCatalogData, options SQLRepositoryOptions) (*CatalogRepository, error) {
	db, _, store, err := internalinfra.OpenSQLStateStore(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return newSQLRepositoryWithStore(ctx, &SQLStateStore{inner: store}, seedData, options, db)
}

func NewSQLRepositoryWithDB(ctx context.Context, db *sql.DB, driverName string, seedData SampleCatalogData, options SQLRepositoryOptions) (*CatalogRepository, error) {
	store, err := internalinfra.PrepareSQLStateStore(ctx, db, driverName)
	if err != nil {
		return nil, err
	}
	return newSQLRepositoryWithStore(ctx, &SQLStateStore{inner: store}, seedData, options, nil)
}

func newSQLRepositoryWithStore(ctx context.Context, store *SQLStateStore, seedData SampleCatalogData, options SQLRepositoryOptions, ownedDB *sql.DB) (*CatalogRepository, error) {
	artifactSource := options.ArtifactSource
	if artifactSource == nil {
		artifactSource = NewStaticArtifactSource(seedData.Artifacts)
	}

	repository := NewCatalogRepository(CatalogRepositoryConfig{
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

func sqlDriverNameForURL(databaseURL string) string {
	return internalinfra.SQLDriverNameForURL(databaseURL)
}
