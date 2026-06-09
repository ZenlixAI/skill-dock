package catalog

import (
	"context"
	"database/sql"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type SQLRepositoryOptions struct {
	ArtifactSource ArtifactSource
	DisableSeed    bool
}

func NewSQLRepository(ctx context.Context, databaseURL string, seedData SampleCatalogData, options SQLRepositoryOptions) (*CatalogRepository, error) {
	driverName := sqlDriverNameForURL(databaseURL)
	db, err := sql.Open(driverName, databaseURL)
	if err != nil {
		return nil, err
	}
	repository, err := NewSQLRepositoryWithDB(ctx, db, driverName, seedData, options)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return repository, nil
}

func NewSQLRepositoryWithDB(ctx context.Context, db *sql.DB, driverName string, seedData SampleCatalogData, options SQLRepositoryOptions) (*CatalogRepository, error) {
	store := NewSQLStateStore(db, driverName)
	if err := store.Migrate(ctx); err != nil {
		return nil, err
	}

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
			return nil, err
		}
	}
	return repository, nil
}

func sqlDriverNameForURL(databaseURL string) string {
	switch {
	case strings.HasPrefix(databaseURL, "postgres://"), strings.HasPrefix(databaseURL, "postgresql://"):
		return "pgx"
	default:
		return "sqlite"
	}
}
