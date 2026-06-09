package persist

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
	"skill_registry/app/observability"
	"skill_registry/business/catalog"
)

type AuditStore interface {
	Append(ctx context.Context, event AuditEvent) error
	List(ctx context.Context, query AuditQuery) (AuditPage, error)
}

type AuditEvent struct {
	Action        string
	RequestID     string
	TraceID       string
	Actor         string
	Role          string
	AuthMechanism string
	AuthDecision  observability.AuthDecision
	Endpoint      string
	Method        string
	PackageName   string
	Version       *string
	Outcome       string
	StatusCode    int
	Details       map[string]any
	CreatedAt     time.Time
}

type AuditQuery struct {
	Action       string
	Actor        string
	AuthDecision string
	PackageName  string
	Version      string
	Outcome      string
	PageSize     int
	Cursor       int
}

type AuditRecord struct {
	ID            string                     `json:"id"`
	RequestID     string                     `json:"requestId"`
	TraceID       string                     `json:"traceId"`
	Action        string                     `json:"action"`
	Actor         string                     `json:"actor"`
	Role          string                     `json:"role"`
	AuthMechanism string                     `json:"authMechanism"`
	AuthDecision  observability.AuthDecision `json:"authDecision"`
	Endpoint      string                     `json:"endpoint"`
	Method        string                     `json:"method"`
	PackageName   string                     `json:"packageName"`
	Version       *string                    `json:"version,omitempty"`
	Outcome       string                     `json:"outcome"`
	StatusCode    int                        `json:"statusCode"`
	Details       map[string]any             `json:"details"`
	CreatedAt     time.Time                  `json:"createdAt"`
}

type AuditPage struct {
	Items    []AuditRecord    `json:"items"`
	PageInfo catalog.PageInfo `json:"pageInfo"`
}

type SQLAuditStore struct {
	db      *sql.DB
	dialect string
}

func NewSQLAuditStore(ctx context.Context, databaseURL string) (*SQLAuditStore, error) {
	driver := auditSQLDriverNameForURL(databaseURL)
	db, err := sql.Open(driver, databaseURL)
	if err != nil {
		return nil, err
	}
	store := newSQLAuditStore(db, auditSQLDialectForDriver(driver))
	if err := store.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func newSQLAuditStore(db *sql.DB, dialect string) *SQLAuditStore {
	return &SQLAuditStore{
		db:      db,
		dialect: dialect,
	}
}

func (s *SQLAuditStore) Append(ctx context.Context, event AuditEvent) error {
	detailsJSON, err := json.Marshal(defaultAuditDetails(event.Details))
	if err != nil {
		return err
	}
	query := auditBindSQL(s.dialect, `
INSERT INTO audit_events (
	id,
	request_id,
	trace_id,
	action,
	actor,
	role,
	auth_mechanism,
	auth_decision,
	endpoint,
	method,
	package_name,
	version,
	outcome,
	status_code,
	details_json,
	created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	_, err = s.db.ExecContext(
		ctx,
		query,
		fmt.Sprintf("aud_%d", time.Now().UTC().UnixNano()),
		strings.TrimSpace(event.RequestID),
		strings.TrimSpace(event.TraceID),
		strings.TrimSpace(event.Action),
		strings.TrimSpace(event.Actor),
		strings.TrimSpace(event.Role),
		strings.TrimSpace(event.AuthMechanism),
		strings.TrimSpace(string(event.AuthDecision)),
		strings.TrimSpace(event.Endpoint),
		strings.TrimSpace(event.Method),
		strings.TrimSpace(event.PackageName),
		nullableString(event.Version),
		strings.TrimSpace(event.Outcome),
		event.StatusCode,
		string(detailsJSON),
		auditTimestamp(event.CreatedAt),
	)
	return err
}

func (s *SQLAuditStore) List(ctx context.Context, query AuditQuery) (AuditPage, error) {
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		return AuditPage{}, catalog.NewError(400, "invalid_request", "pageSize must be less than or equal to 100.", map[string]any{"pageSize": query.PageSize})
	}
	if query.Cursor < 0 {
		return AuditPage{}, catalog.NewError(400, "invalid_request", "cursor must be greater than or equal to 0.", map[string]any{"cursor": query.Cursor})
	}

	queryText := s.listSQL(query)

	rows, err := s.db.QueryContext(
		ctx,
		queryText,
		s.listArgs(query)...,
	)
	if err != nil {
		return AuditPage{}, err
	}
	defer rows.Close()

	var items []AuditRecord
	for rows.Next() {
		var (
			record       AuditRecord
			version      sql.NullString
			detailsJSON  string
			createdAtRaw string
		)
		if err := rows.Scan(
			&record.ID,
			&record.RequestID,
			&record.TraceID,
			&record.Action,
			&record.Actor,
			&record.Role,
			&record.AuthMechanism,
			&record.AuthDecision,
			&record.Endpoint,
			&record.Method,
			&record.PackageName,
			&version,
			&record.Outcome,
			&record.StatusCode,
			&detailsJSON,
			&createdAtRaw,
		); err != nil {
			return AuditPage{}, err
		}
		if version.Valid && strings.TrimSpace(version.String) != "" {
			record.Version = stringPtr(version.String)
		}
		if err := json.Unmarshal([]byte(detailsJSON), &record.Details); err != nil {
			return AuditPage{}, err
		}
		record.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAtRaw)
		if err != nil {
			record.CreatedAt, err = time.Parse(time.RFC3339, createdAtRaw)
			if err != nil {
				return AuditPage{}, err
			}
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return AuditPage{}, err
	}

	hasNextPage := len(items) > query.PageSize
	if hasNextPage {
		items = items[:query.PageSize]
	}

	var nextCursor *string
	if hasNextPage {
		cursor := strconv.Itoa(query.Cursor + len(items))
		nextCursor = &cursor
	}

	return AuditPage{
		Items: items,
		PageInfo: catalog.PageInfo{
			NextCursor:  nextCursor,
			HasNextPage: hasNextPage,
		},
	}, nil
}

func (s *SQLAuditStore) listSQL(query AuditQuery) string {
	parts := []string{`
SELECT
	id,
	request_id,
	trace_id,
	action,
	actor,
	role,
	auth_mechanism,
	auth_decision,
	endpoint,
	method,
	package_name,
	version,
	outcome,
	status_code,
	details_json,
	created_at
FROM audit_events
WHERE 1 = 1`}
	if strings.TrimSpace(query.Action) != "" {
		parts = append(parts, `AND action = ?`)
	}
	if strings.TrimSpace(query.Actor) != "" {
		parts = append(parts, `AND actor = ?`)
	}
	if strings.TrimSpace(query.AuthDecision) != "" {
		parts = append(parts, `AND auth_decision = ?`)
	}
	if strings.TrimSpace(query.PackageName) != "" {
		parts = append(parts, `AND package_name = ?`)
	}
	if strings.TrimSpace(query.Version) != "" {
		parts = append(parts, `AND version = ?`)
	}
	if strings.TrimSpace(query.Outcome) != "" {
		parts = append(parts, `AND outcome = ?`)
	}
	parts = append(parts, `ORDER BY created_at DESC`, `LIMIT ?`, `OFFSET ?`)
	return auditBindSQL(s.dialect, strings.Join(parts, "\n"))
}

func (s *SQLAuditStore) listArgs(query AuditQuery) []any {
	args := make([]any, 0, 8)
	if strings.TrimSpace(query.Action) != "" {
		args = append(args, query.Action)
	}
	if strings.TrimSpace(query.Actor) != "" {
		args = append(args, query.Actor)
	}
	if strings.TrimSpace(query.AuthDecision) != "" {
		args = append(args, query.AuthDecision)
	}
	if strings.TrimSpace(query.PackageName) != "" {
		args = append(args, query.PackageName)
	}
	if strings.TrimSpace(query.Version) != "" {
		args = append(args, query.Version)
	}
	if strings.TrimSpace(query.Outcome) != "" {
		args = append(args, query.Outcome)
	}
	args = append(args, query.PageSize+1, query.Cursor)
	return args
}

func (s *SQLAuditStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *SQLAuditStore) migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS audit_events (
			id TEXT PRIMARY KEY,
			request_id TEXT NOT NULL,
			trace_id TEXT NOT NULL DEFAULT '',
			action TEXT NOT NULL,
			actor TEXT NOT NULL,
			role TEXT NOT NULL,
			auth_mechanism TEXT NOT NULL,
			auth_decision TEXT NOT NULL DEFAULT '',
			endpoint TEXT NOT NULL,
			method TEXT NOT NULL,
			package_name TEXT NOT NULL,
			version TEXT,
			outcome TEXT NOT NULL,
			status_code INTEGER NOT NULL,
			details_json TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_events_action_created_at ON audit_events(action, created_at)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := s.db.ExecContext(ctx, `ALTER TABLE audit_events ADD COLUMN trace_id TEXT NOT NULL DEFAULT ''`); err != nil && !isIgnorableAddColumnError(err) {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `ALTER TABLE audit_events ADD COLUMN auth_decision TEXT NOT NULL DEFAULT ''`); err != nil && !isIgnorableAddColumnError(err) {
		return err
	}
	return nil
}

func auditSQLDriverNameForURL(databaseURL string) string {
	lower := strings.ToLower(databaseURL)
	switch {
	case strings.HasPrefix(lower, "postgres://"), strings.HasPrefix(lower, "postgresql://"):
		return "pgx"
	default:
		return "sqlite"
	}
}

func auditSQLDialectForDriver(driver string) string {
	if driver == "pgx" {
		return "postgres"
	}
	return "sqlite"
}

func auditBindSQL(dialect string, query string) string {
	if dialect != "postgres" {
		return query
	}
	index := 1
	var builder strings.Builder
	for _, char := range query {
		if char == '?' {
			builder.WriteString(fmt.Sprintf("$%d", index))
			index++
			continue
		}
		builder.WriteRune(char)
	}
	return builder.String()
}

func auditTimestamp(value time.Time) string {
	if value.IsZero() {
		value = time.Now().UTC()
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func nullableString(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return strings.TrimSpace(*value)
}

func defaultAuditDetails(details map[string]any) map[string]any {
	if details == nil {
		return map[string]any{}
	}
	return details
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func isIgnorableAddColumnError(err error) bool {
	if err == nil {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate column") || strings.Contains(message, "already exists")
}
