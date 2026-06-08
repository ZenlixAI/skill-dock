package persist

import (
	"context"
	"database/sql"
	"encoding/json"
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
	SaveValidationCache(ctx context.Context, workspaceID string, cache WorkspaceValidationCache, updatedAt time.Time) error
	SavePackageMetadata(ctx context.Context, workspaceID string, packageMetadataJSON []byte, updatedAt time.Time) error
	LoadBundle(ctx context.Context, workspaceID string) (string, error)
	ListExpired(ctx context.Context, cutoff time.Time) ([]WorkspaceRecord, error)
	DeleteWorkspace(ctx context.Context, workspaceID string) error
	DeleteExpired(ctx context.Context, cutoff time.Time) error
}

type WorkspaceValidationCache struct {
	PackageName    string   `json:"packageName"`
	Version        string   `json:"version"`
	ManifestDigest string   `json:"manifestDigest,omitempty"`
	Components     []string `json:"components,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
	Errors         []string `json:"errors,omitempty"`
}

type WorkspaceRecord struct {
	WorkspaceID         string
	WorkspacePath       string
	BundlePath          string
	ValidationCache     *WorkspaceValidationCache
	PackageMetadataJSON []byte
	CreatedAt           time.Time
	UpdatedAt           time.Time
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
			validation_cache_json TEXT,
			package_metadata_json TEXT,
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

	hasValidationCache, err := s.workspaceColumnExists(ctx, "validation_cache_json")
	if err != nil {
		return err
	}
	if !hasValidationCache {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE packaging_workspaces ADD COLUMN validation_cache_json TEXT`); err != nil {
			return err
		}
	}

	hasPackageMetadata, err := s.workspaceColumnExists(ctx, "package_metadata_json")
	if err != nil {
		return err
	}
	if !hasPackageMetadata {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE packaging_workspaces ADD COLUMN package_metadata_json TEXT`); err != nil {
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
	validation_cache_json,
	package_metadata_json,
	created_at,
	updated_at
) VALUES (?, ?, NULL, NULL, NULL, ?, ?)
ON CONFLICT(workspace_id) DO UPDATE SET
	workspace_path = excluded.workspace_path,
	bundle_path = NULL,
	validation_cache_json = NULL,
	package_metadata_json = NULL,
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
	validation_cache_json,
	package_metadata_json,
	created_at,
	updated_at
) VALUES (?, ?, NULL, ?, NULL, NULL, NULL, ?, ?)
ON CONFLICT(workspace_id) DO UPDATE SET
	workspace_path = excluded.workspace_path,
	bundle_path = NULL,
	validation_cache_json = NULL,
	package_metadata_json = NULL,
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
SELECT workspace_path, bundle_path, validation_cache_json, package_metadata_json, created_at, updated_at
FROM packaging_workspaces
WHERE workspace_id = ?`)
	var (
		workspacePath       sql.NullString
		bundlePath          sql.NullString
		validationCacheJSON sql.NullString
		packageMetadataJSON sql.NullString
		createdAtRaw        string
		updatedAtRaw        string
	)
	err := s.db.QueryRowContext(ctx, query, workspaceID).Scan(
		&workspacePath,
		&bundlePath,
		&validationCacheJSON,
		&packageMetadataJSON,
		&createdAtRaw,
		&updatedAtRaw,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return WorkspaceRecord{}, ErrWorkspaceRecordNotFound
	}
	if err != nil {
		return WorkspaceRecord{}, err
	}
	return scanWorkspaceRecord(workspaceID, workspacePath, bundlePath, validationCacheJSON, packageMetadataJSON, createdAtRaw, updatedAtRaw)
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

func (s *SQLWorkspaceStore) SaveValidationCache(ctx context.Context, workspaceID string, cache WorkspaceValidationCache, updatedAt time.Time) error {
	content, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	query := bindSQL(s.dialect, `
UPDATE packaging_workspaces
SET validation_cache_json = ?, updated_at = ?
WHERE workspace_id = ?`)
	result, err := s.db.ExecContext(
		ctx,
		query,
		string(content),
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

func (s *SQLWorkspaceStore) SavePackageMetadata(ctx context.Context, workspaceID string, packageMetadataJSON []byte, updatedAt time.Time) error {
	query := bindSQL(s.dialect, `
UPDATE packaging_workspaces
SET package_metadata_json = ?, updated_at = ?
WHERE workspace_id = ?`)
	result, err := s.db.ExecContext(
		ctx,
		query,
		string(packageMetadataJSON),
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
SELECT workspace_id, workspace_path, bundle_path, validation_cache_json, package_metadata_json, created_at, updated_at
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
			workspaceID         string
			workspacePath       sql.NullString
			bundlePath          sql.NullString
			validationCacheJSON sql.NullString
			packageMetadataJSON sql.NullString
			createdAtRaw        string
			updatedAtRaw        string
		)
		if err := rows.Scan(&workspaceID, &workspacePath, &bundlePath, &validationCacheJSON, &packageMetadataJSON, &createdAtRaw, &updatedAtRaw); err != nil {
			return nil, err
		}
		record, err := scanWorkspaceRecord(workspaceID, workspacePath, bundlePath, validationCacheJSON, packageMetadataJSON, createdAtRaw, updatedAtRaw)
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

func scanWorkspaceRecord(workspaceID string, workspacePath sql.NullString, bundlePath sql.NullString, validationCacheJSON sql.NullString, packageMetadataJSON sql.NullString, createdAtRaw string, updatedAtRaw string) (WorkspaceRecord, error) {
	if !workspacePath.Valid || strings.TrimSpace(workspacePath.String) == "" {
		return WorkspaceRecord{}, ErrWorkspaceRecordNotFound
	}

	record := WorkspaceRecord{
		WorkspaceID:   workspaceID,
		WorkspacePath: workspacePath.String,
		BundlePath:    bundlePath.String,
	}
	if validationCacheJSON.Valid && strings.TrimSpace(validationCacheJSON.String) != "" {
		record.ValidationCache = &WorkspaceValidationCache{}
		if err := json.Unmarshal([]byte(validationCacheJSON.String), record.ValidationCache); err != nil {
			return WorkspaceRecord{}, err
		}
	}
	if packageMetadataJSON.Valid && strings.TrimSpace(packageMetadataJSON.String) != "" {
		record.PackageMetadataJSON = []byte(packageMetadataJSON.String)
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
