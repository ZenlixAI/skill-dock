package persist

import platformpersist "skill_registry/internal/platform/persistence"

var (
	ErrWorkspaceRecordNotFound = platformpersist.ErrWorkspaceRecordNotFound
	ErrWorkspaceBundleNotFound = platformpersist.ErrWorkspaceBundleNotFound
)

type WorkspaceStore = platformpersist.WorkspaceStore
type WorkspaceValidationCache = platformpersist.WorkspaceValidationCache
type WorkspaceRecord = platformpersist.WorkspaceRecord
type SQLWorkspaceStore = platformpersist.SQLWorkspaceStore

var NewSQLWorkspaceStore = platformpersist.NewSQLWorkspaceStore
