package packaging

import (
	"skill_registry/foundation/persist"
	packagingapplication "skill_registry/internal/packaging/application"
)

type Service = packagingapplication.Service

func NewService(tempRoot string) *Service {
	return packagingapplication.NewService(tempRoot)
}

func NewServiceWithWorkspaceStore(tempRoot string, store persist.WorkspaceStore) *Service {
	return packagingapplication.NewServiceWithWorkspaceStore(tempRoot, store)
}
