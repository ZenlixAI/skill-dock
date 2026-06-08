package persist

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var ErrWorkspaceRecordNotFound = errors.New("workspace record not found")
var ErrWorkspaceBundleNotFound = errors.New("workspace bundle not found")

type WorkspaceStore interface {
	SaveWorkspace(ctx context.Context, workspaceID string, workspacePath string, createdAt time.Time) error
	LoadWorkspace(ctx context.Context, workspaceID string) (WorkspaceRecord, error)
	SaveBundle(ctx context.Context, workspaceID string, bundlePath string, updatedAt time.Time) error
	LoadBundle(ctx context.Context, workspaceID string) (string, error)
	ListExpired(ctx context.Context, cutoff time.Time) ([]WorkspaceRecord, error)
	DeleteWorkspace(ctx context.Context, workspaceID string) error
	DeleteExpired(ctx context.Context, cutoff time.Time) error
}

type WorkspaceRecord struct {
	WorkspaceID   string
	WorkspacePath string
	BundlePath    string
	CreatedAt     time.Time
	UpdatedAt     time.Time
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
	store := newSQLWorkspaceStore(db, dialect)
	if err := store.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func newSQLWorkspaceStore(db *sql.DB, dialect string) *SQLWorkspaceStore {
	return &SQLWorkspaceStore{db: db, dialect: dialect}
}

func (s *SQLWorkspaceStore) Migrate(ctx context.Context) error {
	for _, statement := range workspaceMigrationStatements(s.dialect) {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return s.ensureManagedPathColumns(ctx)
}

func workspaceMigrationStatements(dialect string) []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS packaging_workspaces (
			workspace_id TEXT NOT NULL PRIMARY KEY,
			workspace_path TEXT NOT NULL,
			bundle_path TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_packaging_workspaces_updated_at ON packaging_workspaces(updated_at)`,
	}
}

func (s *SQLWorkspaceStore) ensureManagedPathColumns(ctx context.Context) error {
	hasWorkspacePath, err := s.workspaceColumnExists(ctx, "workspace_path")
	if err != nil {
		return err
	}
	if !hasWorkspacePath {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE packaging_workspaces ADD COLUMN workspace_path TEXT`); err != nil {
			return err
		}
	}

	hasBundlePath, err := s.workspaceColumnExists(ctx, "bundle_path")
	if err != nil {
		return err
	}
	if !hasBundlePath {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE packaging_workspaces ADD COLUMN bundle_path TEXT`); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLWorkspaceStore) workspaceColumnExists(ctx context.Context, columnName string) (bool, error) {
	switch s.dialect {
	case "pgx":
		const query = `
SELECT COUNT(*)
FROM information_schema.columns
WHERE table_schema = current_schema()
  AND table_name = 'packaging_workspaces'
  AND column_name = ?`
		var count int
		if err := s.db.QueryRowContext(ctx, bindSQL(s.dialect, query), columnName).Scan(&count); err != nil {
			return false, err
		}
		return count > 0, nil
	default:
		rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(packaging_workspaces)`)
		if err != nil {
			return false, err
		}
		defer rows.Close()

		for rows.Next() {
			var (
				cid        int
				name       string
				dataType   string
				notNull    int
				defaultVal sql.NullString
				primaryKey int
			)
			if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultVal, &primaryKey); err != nil {
				return false, err
			}
			if name == columnName {
				return true, nil
			}
		}
		if err := rows.Err(); err != nil {
			return false, err
		}
		return false, nil
	}
}

func (s *SQLWorkspaceStore) SaveWorkspace(ctx context.Context, workspaceID string, workspacePath string, createdAt time.Time) error {
	hasLegacySnapshot, err := s.workspaceColumnExists(ctx, "workspace_snapshot")
	if err != nil {
		return err
	}

	query := `
INSERT INTO packaging_workspaces (
	workspace_id,
	workspace_path,
	bundle_path,
	created_at,
	updated_at
) VALUES (?, ?, NULL, ?, ?)
ON CONFLICT(workspace_id) DO UPDATE SET
	workspace_path = excluded.workspace_path,
	bundle_path = NULL,
	updated_at = excluded.updated_at`
	args := []any{
		workspaceID,
		workspacePath,
		createdAt.UTC().Format(time.RFC3339),
		createdAt.UTC().Format(time.RFC3339),
	}
	if hasLegacySnapshot {
		query = `
INSERT INTO packaging_workspaces (
	workspace_id,
	workspace_snapshot,
	bundle_content,
	workspace_path,
	bundle_path,
	created_at,
	updated_at
) VALUES (?, ?, NULL, ?, NULL, ?, ?)
ON CONFLICT(workspace_id) DO UPDATE SET
	workspace_path = excluded.workspace_path,
	bundle_path = NULL,
	updated_at = excluded.updated_at`
		args = []any{
			workspaceID,
			[]byte{},
			workspacePath,
			createdAt.UTC().Format(time.RFC3339),
			createdAt.UTC().Format(time.RFC3339),
		}
	}

	_, err = s.db.ExecContext(
		ctx,
		bindSQL(s.dialect, query),
		args...,
	)
	return err
}

func (s *SQLWorkspaceStore) LoadWorkspace(ctx context.Context, workspaceID string) (WorkspaceRecord, error) {
	query := bindSQL(s.dialect, `
SELECT workspace_path, bundle_path, created_at, updated_at
FROM packaging_workspaces
WHERE workspace_id = ?`)
	var (
		workspacePath sql.NullString
		bundlePath    sql.NullString
		createdAtRaw  string
		updatedAtRaw  string
	)
	err := s.db.QueryRowContext(ctx, query, workspaceID).Scan(
		&workspacePath,
		&bundlePath,
		&createdAtRaw,
		&updatedAtRaw,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return WorkspaceRecord{}, ErrWorkspaceRecordNotFound
	}
	if err != nil {
		return WorkspaceRecord{}, err
	}
	return scanWorkspaceRecord(workspaceID, workspacePath, bundlePath, createdAtRaw, updatedAtRaw)
}

func (s *SQLWorkspaceStore) SaveBundle(ctx context.Context, workspaceID string, bundlePath string, updatedAt time.Time) error {
	query := bindSQL(s.dialect, `
UPDATE packaging_workspaces
SET bundle_path = ?, updated_at = ?
WHERE workspace_id = ?`)
	result, err := s.db.ExecContext(
		ctx,
		query,
		bundlePath,
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

func (s *SQLWorkspaceStore) LoadBundle(ctx context.Context, workspaceID string) (string, error) {
	query := bindSQL(s.dialect, `
SELECT bundle_path
FROM packaging_workspaces
WHERE workspace_id = ?`)
	var bundlePath sql.NullString
	err := s.db.QueryRowContext(ctx, query, workspaceID).Scan(&bundlePath)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrWorkspaceRecordNotFound
	}
	if err != nil {
		return "", err
	}
	if !bundlePath.Valid || strings.TrimSpace(bundlePath.String) == "" {
		return "", ErrWorkspaceBundleNotFound
	}
	return bundlePath.String, nil
}

func (s *SQLWorkspaceStore) ListExpired(ctx context.Context, cutoff time.Time) ([]WorkspaceRecord, error) {
	query := bindSQL(s.dialect, `
SELECT workspace_id, workspace_path, bundle_path, created_at, updated_at
FROM packaging_workspaces
WHERE updated_at < ?
ORDER BY updated_at ASC`)
	rows, err := s.db.QueryContext(ctx, query, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []WorkspaceRecord{}
	for rows.Next() {
		var (
			workspaceID   string
			workspacePath sql.NullString
			bundlePath    sql.NullString
			createdAtRaw  string
			updatedAtRaw  string
		)
		if err := rows.Scan(&workspaceID, &workspacePath, &bundlePath, &createdAtRaw, &updatedAtRaw); err != nil {
			return nil, err
		}
		record, err := scanWorkspaceRecord(workspaceID, workspacePath, bundlePath, createdAtRaw, updatedAtRaw)
		if errors.Is(err, ErrWorkspaceRecordNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func scanWorkspaceRecord(workspaceID string, workspacePath sql.NullString, bundlePath sql.NullString, createdAtRaw string, updatedAtRaw string) (WorkspaceRecord, error) {
	if !workspacePath.Valid || strings.TrimSpace(workspacePath.String) == "" {
		return WorkspaceRecord{}, ErrWorkspaceRecordNotFound
	}

	record := WorkspaceRecord{
		WorkspaceID:   workspaceID,
		WorkspacePath: workspacePath.String,
		BundlePath:    bundlePath.String,
	}

	var err error
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
