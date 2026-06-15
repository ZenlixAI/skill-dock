package persist

import (
	"context"
	"database/sql"

	platformpersist "skill_registry/internal/platform/persistence"
)

type AuditStore = platformpersist.AuditStore
type AuditEvent = platformpersist.AuditEvent
type AuditQuery = platformpersist.AuditQuery
type AuditRecord = platformpersist.AuditRecord
type AuditPage = platformpersist.AuditPage
type SQLAuditStore = platformpersist.SQLAuditStore

var NewSQLAuditStore = platformpersist.NewSQLAuditStore

func newSQLAuditStore(db *sql.DB, dialect string) *SQLAuditStore {
	return platformpersist.NewSQLAuditStoreWithDB(db, dialect)
}

func auditListSQLForTest(store *SQLAuditStore, query AuditQuery) string {
	return platformpersist.AuditListSQLForTest(store, query)
}

func auditListArgsForTest(store *SQLAuditStore, query AuditQuery) []any {
	return platformpersist.AuditListArgsForTest(store, query)
}

func _auditStoreShimContext(_ context.Context) {}
