package persist

import platformpersist "skill_registry/internal/platform/persistence"

type AuditStore = platformpersist.AuditStore
type AuditEvent = platformpersist.AuditEvent
type AuditQuery = platformpersist.AuditQuery
type AuditRecord = platformpersist.AuditRecord
type AuditPage = platformpersist.AuditPage
type SQLAuditStore = platformpersist.SQLAuditStore

var NewSQLAuditStore = platformpersist.NewSQLAuditStore
