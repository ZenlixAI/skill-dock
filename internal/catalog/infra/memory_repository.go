package infra

import "skill_registry/business/catalog"

type MemoryRepositoryOptions = catalog.MemoryRepositoryOptions

func NewMemoryRepository(data catalog.SampleCatalogData, options ...MemoryRepositoryOptions) *catalog.CatalogRepository {
	return catalog.NewMemoryRepository(data, options...)
}
