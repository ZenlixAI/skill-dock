package catalog

import (
	internalapp "skill_registry/internal/catalog/application"
	internaldomain "skill_registry/internal/catalog/domain"
)

type ReadRepository = internaldomain.ReadRepository
type WriteRepository = internaldomain.WriteRepository
type Repository = internaldomain.Repository

type ApplicationServiceProvider interface {
	CatalogReadService() *internalapp.ReadService
	CatalogWriteService() *internalapp.WriteService
}
