package persist

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrWorkspaceRecordNotFound = errors.New("workspace record not found")
var ErrWorkspaceBundleNotFound = errors.New("workspace bundle not found")

type WorkspaceStore interface {
	SaveWorkspace(ctx context.Context, workspaceID string, snapshot []byte, createdAt time.Time) error
	LoadWorkspace(ctx context.Context, workspaceID string) (WorkspaceRecord, error)
	SaveBundle(ctx context.Context, workspaceID string, bundle []byte, updatedAt time.Time) error
	LoadBundle(ctx context.Context, workspaceID string) ([]byte, error)
	DeleteWorkspace(ctx context.Context, workspaceID string) error
	DeleteExpired(ctx context.Context, cutoff time.Time) error
}

type WorkspaceRecord struct {
	WorkspaceID string
	Snapshot    []byte
	Bundle      []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SQLWorkspaceStore struct {
	db      *sql.DB
	dialect string
}

func NewSQLWorkspaceStore(ctx context.Context, databaseURL string) (*SQLWorkspaceStore, error) {
	dialect := sqlDriverNameForURL(databaseURL)
	db, err := sql.Open(dialect, databaseURL)
	if err != nil {
		return nil, err
	}
	store := &SQLWorkspaceStore{db: db, dialect: dialect}
	if err := store.Migrate(ctx); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *SQLWorkspaceStore) Migrate(ctx context.Context) error {
	for _, statement := range workspaceMigrationStatements(s.dialect) {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func workspaceMigrationStatements(dialect string) []string {
	binaryType := "BLOB"
	if dialect == "pgx" {
		binaryType = "BYTEA"
	}
	return []string{
		`CREATE TABLE IF NOT EXISTS packaging_workspaces (
			workspace_id TEXT NOT NULL PRIMARY KEY,
			workspace_snapshot ` + binaryType + ` NOT NULL,
			bundle_content ` + binaryType + `,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_packaging_workspaces_updated_at ON packaging_workspaces(updated_at)`,
	}
}

func (s *SQLWorkspaceStore) SaveWorkspace(ctx context.Context, workspaceID string, snapshot []byte, createdAt time.Time) error {
	query := bindSQL(s.dialect, `
INSERT INTO packaging_workspaces (
	workspace_id,
	workspace_snapshot,
	bundle_content,
	created_at,
	updated_at
) VALUES (?, ?, NULL, ?, ?)
ON CONFLICT(workspace_id) DO UPDATE SET
	workspace_snapshot = excluded.workspace_snapshot,
	bundle_content = NULL,
	updated_at = excluded.updated_at`)
	_, err := s.db.ExecContext(
		ctx,
		query,
		workspaceID,
		snapshot,
		createdAt.UTC().Format(time.RFC3339),
		createdAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLWorkspaceStore) LoadWorkspace(ctx context.Context, workspaceID string) (WorkspaceRecord, error) {
	query := bindSQL(s.dialect, `
SELECT workspace_snapshot, bundle_content, created_at, updated_at
FROM packaging_workspaces
WHERE workspace_id = ?`)
	var (
		record       WorkspaceRecord
		createdAtRaw string
		updatedAtRaw string
	)
	err := s.db.QueryRowContext(ctx, query, workspaceID).Scan(
		&record.Snapshot,
		&record.Bundle,
		&createdAtRaw,
		&updatedAtRaw,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return WorkspaceRecord{}, ErrWorkspaceRecordNotFound
	}
	if err != nil {
		return WorkspaceRecord{}, err
	}
	record.WorkspaceID = workspaceID
	record.CreatedAt, err = time.Parse(time.RFC3339, createdAtRaw)
	if err != nil {
		return WorkspaceRecord{}, err
	}
	record.UpdatedAt, err = time.Parse(time.RFC3339, updatedAtRaw)
	if err != nil {
		return WorkspaceRecord{}, err
	}
	return record, nil
}

func (s *SQLWorkspaceStore) SaveBundle(ctx context.Context, workspaceID string, bundle []byte, updatedAt time.Time) error {
	query := bindSQL(s.dialect, `
UPDATE packaging_workspaces
SET bundle_content = ?, updated_at = ?
WHERE workspace_id = ?`)
	result, err := s.db.ExecContext(
		ctx,
		query,
		bundle,
		updatedAt.UTC().Format(time.RFC3339),
		workspaceID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrWorkspaceRecordNotFound
	}
	return nil
}

func (s *SQLWorkspaceStore) LoadBundle(ctx context.Context, workspaceID string) ([]byte, error) {
	query := bindSQL(s.dialect, `
SELECT bundle_content
FROM packaging_workspaces
WHERE workspace_id = ?`)
	var bundle []byte
	err := s.db.QueryRowContext(ctx, query, workspaceID).Scan(&bundle)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrWorkspaceRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(bundle) == 0 {
		return nil, ErrWorkspaceBundleNotFound
	}
	return bundle, nil
}

func (s *SQLWorkspaceStore) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	query := bindSQL(s.dialect, `DELETE FROM packaging_workspaces WHERE workspace_id = ?`)
	_, err := s.db.ExecContext(ctx, query, workspaceID)
	return err
}

func (s *SQLWorkspaceStore) DeleteExpired(ctx context.Context, cutoff time.Time) error {
	query := bindSQL(s.dialect, `DELETE FROM packaging_workspaces WHERE updated_at < ?`)
	_, err := s.db.ExecContext(ctx, query, cutoff.UTC().Format(time.RFC3339))
	return err
}
