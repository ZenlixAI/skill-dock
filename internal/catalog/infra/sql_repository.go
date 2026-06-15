package infra

import (
	"context"
	"database/sql"

	"skill_registry/business/catalog"
)

type SQLRepositoryOptions = catalog.SQLRepositoryOptions

func NewSQLRepository(ctx context.Context, databaseURL string, seedData catalog.SampleCatalogData, options SQLRepositoryOptions) (*catalog.CatalogRepository, error) {
	return catalog.NewSQLRepository(ctx, databaseURL, seedData, options)
}

func NewSQLRepositoryWithDB(ctx context.Context, db *sql.DB, driverName string, seedData catalog.SampleCatalogData, options SQLRepositoryOptions) (*catalog.CatalogRepository, error) {
	return catalog.NewSQLRepositoryWithDB(ctx, db, driverName, seedData, options)
}
