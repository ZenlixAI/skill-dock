package packaging

import (
	packagingapplication "skill_registry/internal/packaging/application"
	persist "skill_registry/internal/platform/persistence"
)

type Service = packagingapplication.Service

func NewService(tempRoot string) *Service {
	return packagingapplication.NewService(tempRoot)
}

func NewServiceWithWorkspaceStore(tempRoot string, store persist.WorkspaceStore) *Service {
	return packagingapplication.NewServiceWithWorkspaceStore(tempRoot, store)
}
