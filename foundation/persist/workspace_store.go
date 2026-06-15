package persist

import (
	"context"
	"database/sql"
	"time"

	platformpersist "skill_registry/internal/platform/persistence"
)

var (
	ErrWorkspaceRecordNotFound = platformpersist.ErrWorkspaceRecordNotFound
	ErrWorkspaceBundleNotFound = platformpersist.ErrWorkspaceBundleNotFound
)

type WorkspaceStore = platformpersist.WorkspaceStore
type WorkspaceValidationCache = platformpersist.WorkspaceValidationCache
type WorkspaceRecord = platformpersist.WorkspaceRecord
type SQLWorkspaceStore = platformpersist.SQLWorkspaceStore

var NewSQLWorkspaceStore = platformpersist.NewSQLWorkspaceStore

func newSQLWorkspaceStore(db *sql.DB, dialect string) *SQLWorkspaceStore {
	return platformpersist.NewSQLWorkspaceStoreWithDB(db, dialect)
}

func workspaceMigrationStatements(dialect string) []string {
	return platformpersist.WorkspaceMigrationStatements(dialect)
}

func _workspaceStoreShimContext(_ context.Context, _ time.Time) {}
