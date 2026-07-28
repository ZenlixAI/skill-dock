package infra

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	. "skill_registry/internal/catalog/domain"
)

type SQLStateStore struct {
	db      *sql.DB
	dialect string
}

func NewSQLStateStore(db *sql.DB, dialect string) *SQLStateStore {
	return &SQLStateStore{db: db, dialect: dialect}
}

func (s *SQLStateStore) Migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS skills (
			package_name TEXT PRIMARY KEY,
			revision INTEGER NOT NULL DEFAULT 0,
			display_name TEXT NOT NULL,
			description TEXT NOT NULL,
			source_type TEXT NOT NULL,
			status TEXT NOT NULL,
			latest_version TEXT,
			latest_version_status TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			curation_reason TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS skill_versions (
			package_name TEXT NOT NULL,
			version TEXT NOT NULL,
			version_status TEXT NOT NULL,
			install_policy TEXT NOT NULL,
			published_at TEXT NOT NULL,
			npm_registry TEXT NOT NULL,
			tarball_url TEXT NOT NULL,
			integrity TEXT NOT NULL,
			shasum TEXT NOT NULL,
			dist_tags_json TEXT NOT NULL,
			skill_manifest_digest TEXT NOT NULL,
			components_json TEXT NOT NULL,
			PRIMARY KEY (package_name, version)
		)`,
		`CREATE TABLE IF NOT EXISTS governance_records (
			id TEXT PRIMARY KEY,
			target_type TEXT NOT NULL,
			package_name TEXT NOT NULL,
			version TEXT,
			action TEXT NOT NULL,
			reason TEXT NOT NULL,
			replacement_recommendation TEXT,
			changed_by TEXT NOT NULL,
			changed_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_skill_versions_package_name ON skill_versions(package_name)`,
		`CREATE INDEX IF NOT EXISTS idx_governance_package_name ON governance_records(package_name)`,
		`CREATE INDEX IF NOT EXISTS idx_skills_updated_at ON skills(updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_skills_curation_reason ON skills(curation_reason)`,
		`CREATE INDEX IF NOT EXISTS idx_governance_changed_at ON governance_records(changed_at)`,
	}

	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if _, err := s.db.ExecContext(ctx, `ALTER TABLE skills ADD COLUMN revision INTEGER NOT NULL DEFAULT 0`); err != nil && !isIgnorableCatalogAddColumnError(err) {
		return err
	}
	return nil
}

func (s *SQLStateStore) IsEmpty(ctx context.Context) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM skills`).Scan(&count); err != nil {
		return false, err
	}
	return count == 0, nil
}

func (s *SQLStateStore) GetAggregate(ctx context.Context, packageName string) (AgentSkillAggregate, error) {
	query := bindSQL(s.dialect, `
SELECT
	revision,
	package_name,
	display_name,
	description,
	source_type,
	status,
	latest_version,
	latest_version_status,
	created_at,
	updated_at,
	curation_reason
FROM skills
WHERE package_name = ?`)

	var (
		revision            int64
		loadedPackageName   string
		displayName         string
		description         string
		sourceType          string
		status              string
		latestVersion       sql.NullString
		latestVersionStatus sql.NullString
		createdAtRaw        string
		updatedAtRaw        string
		curationReason      string
	)
	err := s.db.QueryRowContext(ctx, query, packageName).Scan(
		&revision,
		&loadedPackageName,
		&displayName,
		&description,
		&sourceType,
		&status,
		&latestVersion,
		&latestVersionStatus,
		&createdAtRaw,
		&updatedAtRaw,
		&curationReason,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return AgentSkillAggregate{}, ErrAgentSkillAggregateNotFound
	}
	if err != nil {
		return AgentSkillAggregate{}, err
	}

	createdAt, err := time.Parse(time.RFC3339, createdAtRaw)
	if err != nil {
		return AgentSkillAggregate{}, err
	}
	updatedAt, err := time.Parse(time.RFC3339, updatedAtRaw)
	if err != nil {
		return AgentSkillAggregate{}, err
	}

	versions, err := s.loadVersionsByPackage(ctx, packageName)
	if err != nil {
		return AgentSkillAggregate{}, err
	}
	governance, err := s.loadGovernanceByPackage(ctx, packageName)
	if err != nil {
		return AgentSkillAggregate{}, err
	}

	return AgentSkillAggregate{
		Revision: uint64(revision),
		Skill: SkillSnapshot{
			Detail: SkillDetail{
				SkillSummary: SkillSummary{
					PackageName:         loadedPackageName,
					DisplayName:         displayName,
					Description:         description,
					SourceType:          SourceType(sourceType),
					Status:              SkillStatus(status),
					LatestVersion:       nullStringPtr(latestVersion),
					LatestVersionStatus: nullStringPtr(latestVersionStatus),
					UpdatedAt:           updatedAt,
				},
				CreatedAt: createdAt,
			},
			CurationReason: curationReason,
			Versions:       versions,
		},
		Governance: governance,
	}, nil
}

func (s *SQLStateStore) PutAggregate(ctx context.Context, aggregate AgentSkillAggregate, expectedRevision uint64) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	currentRevision, exists, err := s.loadAggregateRevisionTx(ctx, tx, aggregate.Skill.Detail.PackageName)
	if err != nil {
		return err
	}
	switch {
	case exists && currentRevision != expectedRevision:
		return ErrAggregateRevisionConflict
	case !exists && expectedRevision != 0:
		return ErrAggregateRevisionConflict
	}

	aggregate = cloneAggregate(aggregate)
	if exists {
		if err = s.updateAggregateTx(ctx, tx, aggregate, expectedRevision+1); err != nil {
			return err
		}
	} else if err = s.insertAggregateTx(ctx, tx, aggregate, expectedRevision+1); err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

func (s *SQLStateStore) ListSkills(ctx context.Context, query SkillQuery) (SkillsPage, error) {
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		return SkillsPage{}, NewError(400, "invalid_request", "pageSize must be less than or equal to 100.", map[string]any{"pageSize": query.PageSize})
	}
	if err := validateCursor(query.Cursor); err != nil {
		return SkillsPage{}, err
	}

	sortKey, err := normalizedSkillsSort(query.Sort, query.Query)
	if err != nil {
		return SkillsPage{}, err
	}

	rows, err := s.db.QueryContext(ctx, bindSQL(s.dialect, s.listSkillsSQL(query, sortKey)), s.listSkillsArgs(query, sortKey)...)
	if err != nil {
		return SkillsPage{}, err
	}
	defer rows.Close()

	items := make([]SkillSummary, 0, query.PageSize+1)
	for rows.Next() {
		item, err := scanSkillSummary(rows)
		if err != nil {
			return SkillsPage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return SkillsPage{}, err
	}
	if err := s.validateCursorWindow(ctx, s.listSkillsSelectSQL(query), s.listSkillsFilterArgs(query), query.Cursor, len(items)); err != nil {
		return SkillsPage{}, err
	}
	return SkillsPage{Items: pageWindow(items, query.Cursor, query.PageSize), PageInfo: pageInfoForWindow(items, query.Cursor, query.PageSize)}, nil
}

func (s *SQLStateStore) ListCuratedSkills(ctx context.Context, query CuratedSkillQuery) (CuratedSkillsPage, error) {
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		return CuratedSkillsPage{}, NewError(400, "invalid_request", "pageSize must be less than or equal to 100.", map[string]any{"pageSize": query.PageSize})
	}
	if err := validateCursor(query.Cursor); err != nil {
		return CuratedSkillsPage{}, err
	}

	rows, err := s.db.QueryContext(ctx, bindSQL(s.dialect, s.listCuratedSkillsSQL(query)), s.listCuratedSkillsArgs(query)...)
	if err != nil {
		return CuratedSkillsPage{}, err
	}
	defer rows.Close()

	items := make([]CuratedSkillSummary, 0, query.PageSize+1)
	for rows.Next() {
		var (
			latestVersion       sql.NullString
			latestVersionStatus sql.NullString
			updatedAtRaw        string
			item                CuratedSkillSummary
		)
		if err := rows.Scan(
			&item.PackageName,
			&item.DisplayName,
			&item.Description,
			&item.SourceType,
			&item.Status,
			&latestVersion,
			&latestVersionStatus,
			&updatedAtRaw,
			&item.CurationReason,
		); err != nil {
			return CuratedSkillsPage{}, err
		}
		updatedAt, err := time.Parse(time.RFC3339, updatedAtRaw)
		if err != nil {
			return CuratedSkillsPage{}, err
		}
		item.LatestVersion = nullStringPtr(latestVersion)
		item.LatestVersionStatus = nullStringPtr(latestVersionStatus)
		item.UpdatedAt = updatedAt
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return CuratedSkillsPage{}, err
	}
	if err := s.validateCursorWindow(ctx, s.listCuratedSkillsSelectSQL(query), s.listCuratedSkillsFilterArgs(query), query.Cursor, len(items)); err != nil {
		return CuratedSkillsPage{}, err
	}
	return CuratedSkillsPage{Items: pageWindow(items, query.Cursor, query.PageSize), PageInfo: pageInfoForWindow(items, query.Cursor, query.PageSize)}, nil
}

func (s *SQLStateStore) GetSkill(ctx context.Context, packageName string) (SkillDetail, error) {
	query := bindSQL(s.dialect, `
SELECT
	package_name,
	display_name,
	description,
	source_type,
	status,
	latest_version,
	latest_version_status,
	created_at,
	updated_at
FROM skills
WHERE package_name = ?`)

	detail, err := scanSkillDetail(s.db.QueryRowContext(ctx, query, packageName))
	if errors.Is(err, sql.ErrNoRows) {
		return SkillDetail{}, NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}
	if err != nil {
		return SkillDetail{}, err
	}
	return detail, nil
}

func (s *SQLStateStore) ListVersions(ctx context.Context, query VersionQuery) (VersionsPage, error) {
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		return VersionsPage{}, NewError(400, "invalid_request", "pageSize must be less than or equal to 100.", map[string]any{"pageSize": query.PageSize})
	}
	if err := validateCursor(query.Cursor); err != nil {
		return VersionsPage{}, err
	}

	exists, err := s.skillExists(ctx, query.PackageName)
	if err != nil {
		return VersionsPage{}, err
	}
	if !exists {
		return VersionsPage{}, NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": query.PackageName})
	}

	rows, err := s.db.QueryContext(ctx, bindSQL(s.dialect, s.listVersionsSQL(query)), s.listVersionsArgs(query)...)
	if err != nil {
		return VersionsPage{}, err
	}
	defer rows.Close()

	items := make([]SkillVersionSummary, 0, query.PageSize+1)
	for rows.Next() {
		var (
			publishedAtRaw string
			item           SkillVersionSummary
		)
		if err := rows.Scan(
			&item.PackageName,
			&item.Version,
			&item.VersionStatus,
			&item.InstallPolicy,
			&publishedAtRaw,
		); err != nil {
			return VersionsPage{}, err
		}
		publishedAt, err := time.Parse(time.RFC3339, publishedAtRaw)
		if err != nil {
			return VersionsPage{}, err
		}
		item.PublishedAt = publishedAt
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return VersionsPage{}, err
	}
	if err := s.validateCursorWindow(ctx, s.listVersionsSelectSQL(query), s.listVersionsFilterArgs(query), query.Cursor, len(items)); err != nil {
		return VersionsPage{}, err
	}
	return VersionsPage{Items: pageWindow(items, query.Cursor, query.PageSize), PageInfo: pageInfoForWindow(items, query.Cursor, query.PageSize)}, nil
}

func (s *SQLStateStore) GetVersion(ctx context.Context, packageName string, version string) (SkillVersionDetail, error) {
	query := bindSQL(s.dialect, `
SELECT
	package_name,
	version,
	version_status,
	install_policy,
	published_at,
	npm_registry,
	tarball_url,
	integrity,
	shasum,
	dist_tags_json,
	skill_manifest_digest,
	components_json
FROM skill_versions
WHERE package_name = ? AND version = ?`)

	detail, err := scanSkillVersionDetail(s.db.QueryRowContext(ctx, query, packageName, version))
	if errors.Is(err, sql.ErrNoRows) {
		exists, existsErr := s.skillExists(ctx, packageName)
		if existsErr != nil {
			return SkillVersionDetail{}, existsErr
		}
		if !exists {
			return SkillVersionDetail{}, NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
		}
		return SkillVersionDetail{}, NewError(404, "version_not_found", "Version not found.", map[string]any{
			"packageName": packageName,
			"version":     version,
		})
	}
	if err != nil {
		return SkillVersionDetail{}, err
	}
	return detail, nil
}

func (s *SQLStateStore) GetLatestResolvableVersion(ctx context.Context, packageName string) (SkillVersionDetail, error) {
	query := bindSQL(s.dialect, strings.Join([]string{`
SELECT
	package_name,
	version,
	version_status,
	install_policy,
	published_at,
	npm_registry,
	tarball_url,
	integrity,
	shasum,
	dist_tags_json,
	skill_manifest_digest,
	components_json
FROM skill_versions
WHERE package_name = ?
  AND version_status = 'published'
  AND install_policy = 'resolvable'
`, s.versionOrderBySQL(), `LIMIT 1`}, "\n"))

	detail, err := scanSkillVersionDetail(s.db.QueryRowContext(ctx, query, packageName))
	if errors.Is(err, sql.ErrNoRows) {
		return SkillVersionDetail{}, ErrNoResolvableVersion
	}
	if err != nil {
		return SkillVersionDetail{}, err
	}
	return detail, nil
}

func (s *SQLStateStore) ListPackageVersions(ctx context.Context, packageName string) ([]SkillVersionDetail, error) {
	exists, err := s.skillExists(ctx, packageName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}
	return s.loadVersionsByPackage(ctx, packageName)
}

func (s *SQLStateStore) ListGovernanceChanges(ctx context.Context, query GovernanceQuery) (GovernancePage, error) {
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		return GovernancePage{}, NewError(400, "invalid_request", "pageSize must be less than or equal to 100.", map[string]any{"pageSize": query.PageSize})
	}
	if err := validateCursor(query.Cursor); err != nil {
		return GovernancePage{}, err
	}

	rows, err := s.db.QueryContext(ctx, bindSQL(s.dialect, s.listGovernanceSQL(query)), s.listGovernanceArgs(query)...)
	if err != nil {
		return GovernancePage{}, err
	}
	defer rows.Close()

	items := make([]GovernanceRecord, 0, query.PageSize+1)
	for rows.Next() {
		record, err := scanGovernanceRecord(rows)
		if err != nil {
			return GovernancePage{}, err
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return GovernancePage{}, err
	}
	if err := s.validateCursorWindow(ctx, s.listGovernanceSelectSQL(query), s.listGovernanceFilterArgs(query), query.Cursor, len(items)); err != nil {
		return GovernancePage{}, err
	}
	return GovernancePage{Items: pageWindow(items, query.Cursor, query.PageSize), PageInfo: pageInfoForWindow(items, query.Cursor, query.PageSize)}, nil
}

func (s *SQLStateStore) ListPackageGovernance(ctx context.Context, packageName string, targetType GovernanceTargetType, version *string) ([]GovernanceRecord, error) {
	exists, err := s.skillExists(ctx, packageName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, NewError(404, "skill_not_found", "Skill not found.", map[string]any{"packageName": packageName})
	}

	query := GovernanceQuery{
		PackageName: packageName,
		TargetType:  string(targetType),
	}
	if version != nil {
		query.Version = *version
	}

	governanceSQL := strings.Join([]string{s.listGovernanceSelectSQL(query), `ORDER BY changed_at DESC, id ASC`}, "\n")
	rows, err := s.db.QueryContext(ctx, bindSQL(s.dialect, governanceSQL), s.listGovernanceFilterArgs(query)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]GovernanceRecord, 0)
	for rows.Next() {
		record, err := scanGovernanceRecord(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *SQLStateStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *SQLStateStore) loadVersionsByPackage(ctx context.Context, packageName string) ([]SkillVersionDetail, error) {
	rows, err := s.db.QueryContext(ctx, bindSQL(s.dialect, strings.Join([]string{`
SELECT
	package_name,
	version,
	version_status,
	install_policy,
	published_at,
	npm_registry,
	tarball_url,
	integrity,
	shasum,
	dist_tags_json,
	skill_manifest_digest,
	components_json
FROM skill_versions
WHERE package_name = ?
`, s.versionOrderBySQL()}, "\n")), packageName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := make([]SkillVersionDetail, 0)
	for rows.Next() {
		version, err := scanSkillVersionDetail(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

func (s *SQLStateStore) loadGovernanceByPackage(ctx context.Context, packageName string) ([]GovernanceRecord, error) {
	rows, err := s.db.QueryContext(ctx, bindSQL(s.dialect, `
SELECT
	id,
	target_type,
	package_name,
	version,
	action,
	reason,
	replacement_recommendation,
	changed_by,
	changed_at
FROM governance_records
WHERE package_name = ?
ORDER BY changed_at DESC, id ASC`), packageName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]GovernanceRecord, 0)
	for rows.Next() {
		record, err := scanGovernanceRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *SQLStateStore) loadAggregateRevisionTx(ctx context.Context, tx *sql.Tx, packageName string) (uint64, bool, error) {
	var revision int64
	err := tx.QueryRowContext(ctx, bindSQL(s.dialect, `SELECT revision FROM skills WHERE package_name = ?`), packageName).Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return uint64(revision), true, nil
}

func (s *SQLStateStore) updateAggregateTx(ctx context.Context, tx *sql.Tx, aggregate AgentSkillAggregate, revision uint64) error {
	if err := s.updateSkillSnapshotTx(ctx, tx, aggregate, revision); err != nil {
		return err
	}
	if err := s.replaceVersionRowsTx(ctx, tx, aggregate.Skill.Detail.PackageName, aggregate.Skill.Versions); err != nil {
		return err
	}
	return s.syncGovernanceRowsTx(ctx, tx, aggregate.Skill.Detail.PackageName, aggregate.Governance)
}

func (s *SQLStateStore) insertAggregateTx(ctx context.Context, tx *sql.Tx, aggregate AgentSkillAggregate, revision uint64) error {
	insertSkillSQL := bindSQL(s.dialect, `
INSERT INTO skills (
	package_name,
	revision,
	display_name,
	description,
	source_type,
	status,
	latest_version,
	latest_version_status,
	created_at,
	updated_at,
	curation_reason
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	insertVersionSQL := bindSQL(s.dialect, `
INSERT INTO skill_versions (
	package_name,
	version,
	version_status,
	install_policy,
	published_at,
	npm_registry,
	tarball_url,
	integrity,
	shasum,
	dist_tags_json,
	skill_manifest_digest,
	components_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	insertGovernanceSQL := bindSQL(s.dialect, `
INSERT INTO governance_records (
	id,
	target_type,
	package_name,
	version,
	action,
	reason,
	replacement_recommendation,
	changed_by,
	changed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)

	if _, err := tx.ExecContext(
		ctx,
		insertSkillSQL,
		aggregate.Skill.Detail.PackageName,
		revision,
		aggregate.Skill.Detail.DisplayName,
		aggregate.Skill.Detail.Description,
		string(aggregate.Skill.Detail.SourceType),
		string(aggregate.Skill.Detail.Status),
		nullableStringValue(aggregate.Skill.Detail.LatestVersion),
		nullableStringValue(aggregate.Skill.Detail.LatestVersionStatus),
		aggregate.Skill.Detail.CreatedAt.UTC().Format(time.RFC3339),
		aggregate.Skill.Detail.UpdatedAt.UTC().Format(time.RFC3339),
		aggregate.Skill.CurationReason,
	); err != nil {
		return err
	}

	for _, version := range aggregate.Skill.Versions {
		if err := insertVersionRowTx(ctx, tx, insertVersionSQL, version); err != nil {
			return err
		}
	}

	for _, record := range aggregate.Governance {
		if err := insertGovernanceRowTx(ctx, tx, insertGovernanceSQL, record); err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLStateStore) updateSkillSnapshotTx(ctx context.Context, tx *sql.Tx, aggregate AgentSkillAggregate, revision uint64) error {
	updateSkillSQL := bindSQL(s.dialect, `
UPDATE skills
SET
	revision = ?,
	display_name = ?,
	description = ?,
	source_type = ?,
	status = ?,
	latest_version = ?,
	latest_version_status = ?,
	created_at = ?,
	updated_at = ?,
	curation_reason = ?
WHERE package_name = ?`)

	_, err := tx.ExecContext(
		ctx,
		updateSkillSQL,
		revision,
		aggregate.Skill.Detail.DisplayName,
		aggregate.Skill.Detail.Description,
		string(aggregate.Skill.Detail.SourceType),
		string(aggregate.Skill.Detail.Status),
		nullableStringValue(aggregate.Skill.Detail.LatestVersion),
		nullableStringValue(aggregate.Skill.Detail.LatestVersionStatus),
		aggregate.Skill.Detail.CreatedAt.UTC().Format(time.RFC3339),
		aggregate.Skill.Detail.UpdatedAt.UTC().Format(time.RFC3339),
		aggregate.Skill.CurationReason,
		aggregate.Skill.Detail.PackageName,
	)
	return err
}

func (s *SQLStateStore) replaceVersionRowsTx(ctx context.Context, tx *sql.Tx, packageName string, versions []SkillVersionDetail) error {
	if _, err := tx.ExecContext(ctx, bindSQL(s.dialect, `DELETE FROM skill_versions WHERE package_name = ?`), packageName); err != nil {
		return err
	}

	insertVersionSQL := bindSQL(s.dialect, `
INSERT INTO skill_versions (
	package_name,
	version,
	version_status,
	install_policy,
	published_at,
	npm_registry,
	tarball_url,
	integrity,
	shasum,
	dist_tags_json,
	skill_manifest_digest,
	components_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	for _, version := range versions {
		if err := insertVersionRowTx(ctx, tx, insertVersionSQL, version); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLStateStore) syncGovernanceRowsTx(ctx context.Context, tx *sql.Tx, packageName string, records []GovernanceRecord) error {
	existingRecords, err := s.loadGovernanceByPackageTx(ctx, tx, packageName)
	if err != nil {
		return err
	}

	existingByID := make(map[string]GovernanceRecord, len(existingRecords))
	for _, record := range existingRecords {
		existingByID[record.ID] = record
	}

	nextByID := make(map[string]GovernanceRecord, len(records))
	for _, record := range records {
		nextByID[record.ID] = record
	}

	deleteGovernanceSQL := bindSQL(s.dialect, `DELETE FROM governance_records WHERE id = ?`)
	for _, record := range existingRecords {
		if _, ok := nextByID[record.ID]; ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, deleteGovernanceSQL, record.ID); err != nil {
			return err
		}
	}

	insertGovernanceSQL := bindSQL(s.dialect, `
INSERT INTO governance_records (
	id,
	target_type,
	package_name,
	version,
	action,
	reason,
	replacement_recommendation,
	changed_by,
	changed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	updateGovernanceSQL := bindSQL(s.dialect, `
UPDATE governance_records
SET
	target_type = ?,
	package_name = ?,
	version = ?,
	action = ?,
	reason = ?,
	replacement_recommendation = ?,
	changed_by = ?,
	changed_at = ?
WHERE id = ?`)

	for _, record := range records {
		existing, ok := existingByID[record.ID]
		if !ok {
			if err := insertGovernanceRowTx(ctx, tx, insertGovernanceSQL, record); err != nil {
				return err
			}
			continue
		}
		if governanceRecordEqual(existing, record) {
			continue
		}
		if err := updateGovernanceRowTx(ctx, tx, updateGovernanceSQL, record); err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLStateStore) loadGovernanceByPackageTx(ctx context.Context, tx *sql.Tx, packageName string) ([]GovernanceRecord, error) {
	rows, err := tx.QueryContext(ctx, bindSQL(s.dialect, `
SELECT
	id,
	target_type,
	package_name,
	version,
	action,
	reason,
	replacement_recommendation,
	changed_by,
	changed_at
FROM governance_records
WHERE package_name = ?
ORDER BY changed_at DESC, id ASC`), packageName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]GovernanceRecord, 0)
	for rows.Next() {
		record, err := scanGovernanceRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func insertVersionRowTx(ctx context.Context, tx *sql.Tx, insertVersionSQL string, version SkillVersionDetail) error {
	distTagsJSON, err := json.Marshal(version.Registry.DistTags)
	if err != nil {
		return err
	}
	componentsJSON, err := json.Marshal(version.Bundle.Components)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(
		ctx,
		insertVersionSQL,
		version.PackageName,
		version.Version,
		string(version.VersionStatus),
		string(version.InstallPolicy),
		version.PublishedAt.UTC().Format(time.RFC3339Nano),
		version.Registry.NPMRegistry,
		version.Registry.TarballURL,
		version.Registry.Integrity,
		version.Registry.Shasum,
		string(distTagsJSON),
		version.Bundle.SkillManifestDigest,
		string(componentsJSON),
	)
	return err
}

func insertGovernanceRowTx(ctx context.Context, tx *sql.Tx, insertGovernanceSQL string, record GovernanceRecord) error {
	_, err := tx.ExecContext(
		ctx,
		insertGovernanceSQL,
		record.ID,
		string(record.TargetType),
		record.PackageName,
		nullableStringValue(record.Version),
		string(record.Action),
		record.Reason,
		nullableStringValue(record.ReplacementRecommendation),
		record.ChangedBy,
		record.ChangedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func updateGovernanceRowTx(ctx context.Context, tx *sql.Tx, updateGovernanceSQL string, record GovernanceRecord) error {
	_, err := tx.ExecContext(
		ctx,
		updateGovernanceSQL,
		string(record.TargetType),
		record.PackageName,
		nullableStringValue(record.Version),
		string(record.Action),
		record.Reason,
		nullableStringValue(record.ReplacementRecommendation),
		record.ChangedBy,
		record.ChangedAt.UTC().Format(time.RFC3339),
		record.ID,
	)
	return err
}

func governanceRecordEqual(left GovernanceRecord, right GovernanceRecord) bool {
	return left.ID == right.ID &&
		left.TargetType == right.TargetType &&
		left.PackageName == right.PackageName &&
		nullableStringEqual(left.Version, right.Version) &&
		left.Action == right.Action &&
		left.Reason == right.Reason &&
		nullableStringEqual(left.ReplacementRecommendation, right.ReplacementRecommendation) &&
		left.ChangedBy == right.ChangedBy &&
		left.ChangedAt.Equal(right.ChangedAt)
}

func nullableStringEqual(left *string, right *string) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return *left == *right
	}
}

func nullableStringValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func (s *SQLStateStore) skillExists(ctx context.Context, packageName string) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, bindSQL(s.dialect, `SELECT COUNT(*) FROM skills WHERE package_name = ?`), packageName).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SQLStateStore) listSkillsSelectSQL(query SkillQuery) string {
	parts := []string{`
SELECT
	package_name,
	display_name,
	description,
	source_type,
	status,
	latest_version,
	latest_version_status,
	updated_at
FROM skills
WHERE 1 = 1`}
	if !query.IncludeRetired {
		parts = append(parts, `AND status <> 'retired'`)
	}
	if query.PackageName != "" {
		parts = append(parts, `AND package_name = ?`)
	}
	if query.SourceType != "" {
		parts = append(parts, `AND source_type = ?`)
	}
	if len(query.AllowedPackages) > 0 {
		parts = append(parts, allowedPackageFilterSQL(query.AllowedPackages))
	}
	if strings.TrimSpace(query.Query) != "" {
		parts = append(parts, `AND (LOWER(package_name) LIKE ? OR LOWER(display_name) LIKE ? OR LOWER(description) LIKE ?)`)
	}
	return strings.Join(parts, "\n")
}

func (s *SQLStateStore) listSkillsSQL(query SkillQuery, sortKey string) string {
	parts := []string{s.listSkillsSelectSQL(query), skillsOrderBySQL(sortKey, query.Query), `LIMIT ?`, `OFFSET ?`}
	return strings.Join(parts, "\n")
}

func (s *SQLStateStore) listSkillsFilterArgs(query SkillQuery) []any {
	args := make([]any, 0, 5)
	if query.PackageName != "" {
		args = append(args, query.PackageName)
	}
	if query.SourceType != "" {
		args = append(args, query.SourceType)
	}
	args = append(args, allowedPackageFilterArgs(query.AllowedPackages)...)
	if strings.TrimSpace(query.Query) != "" {
		needle := "%" + strings.ToLower(strings.TrimSpace(query.Query)) + "%"
		args = append(args, needle, needle, needle)
	}
	return args
}

func (s *SQLStateStore) listSkillsArgs(query SkillQuery, sortKey string) []any {
	args := s.listSkillsFilterArgs(query)
	needle := "%" + strings.ToLower(strings.TrimSpace(query.Query)) + "%"
	if sortKey == "relevance" && strings.TrimSpace(query.Query) != "" {
		args = append(args, needle, needle, needle)
	}
	args = append(args, query.PageSize+1, query.Cursor)
	return args
}

func (s *SQLStateStore) listCuratedSkillsSelectSQL(query CuratedSkillQuery) string {
	parts := []string{`
SELECT
	package_name,
	display_name,
	description,
	source_type,
	status,
	latest_version,
	latest_version_status,
	updated_at,
	curation_reason
FROM skills
WHERE curation_reason <> '' AND status <> 'retired'`}
	if len(query.AllowedPackages) > 0 {
		parts = append(parts, allowedPackageFilterSQL(query.AllowedPackages))
	}
	return strings.Join(parts, "\n")
}

func (s *SQLStateStore) listCuratedSkillsSQL(query CuratedSkillQuery) string {
	return strings.Join([]string{s.listCuratedSkillsSelectSQL(query), `ORDER BY updated_at DESC`, `LIMIT ?`, `OFFSET ?`}, "\n")
}

func (s *SQLStateStore) listCuratedSkillsFilterArgs(query CuratedSkillQuery) []any {
	return allowedPackageFilterArgs(query.AllowedPackages)
}

func (s *SQLStateStore) listCuratedSkillsArgs(query CuratedSkillQuery) []any {
	args := s.listCuratedSkillsFilterArgs(query)
	args = append(args, query.PageSize+1, query.Cursor)
	return args
}

func (s *SQLStateStore) listVersionsSelectSQL(query VersionQuery) string {
	parts := []string{`
SELECT
	package_name,
	version,
	version_status,
	install_policy,
	published_at
FROM skill_versions
WHERE package_name = ?`}
	if !query.IncludeDeprecated {
		parts = append(parts, `AND version_status <> 'deprecated'`)
	}
	if !query.IncludeYanked {
		parts = append(parts, `AND version_status <> 'yanked'`)
	}
	return strings.Join(parts, "\n")
}

func (s *SQLStateStore) listVersionsSQL(query VersionQuery) string {
	parts := []string{s.listVersionsSelectSQL(query), s.versionOrderBySQL(), `LIMIT ?`, `OFFSET ?`}
	return strings.Join(parts, "\n")
}

func (s *SQLStateStore) listVersionsFilterArgs(query VersionQuery) []any {
	return []any{query.PackageName}
}

func (s *SQLStateStore) listVersionsArgs(query VersionQuery) []any {
	args := s.listVersionsFilterArgs(query)
	args = append(args, query.PageSize+1, query.Cursor)
	return args
}

func (s *SQLStateStore) listGovernanceSelectSQL(query GovernanceQuery) string {
	parts := []string{`
SELECT
	id,
	target_type,
	package_name,
	version,
	action,
	reason,
	replacement_recommendation,
	changed_by,
	changed_at
FROM governance_records
WHERE 1 = 1`}
	if query.PackageName != "" {
		parts = append(parts, `AND package_name = ?`)
	}
	if query.TargetType != "" {
		parts = append(parts, `AND target_type = ?`)
	}
	if query.Version != "" {
		parts = append(parts, `AND version = ?`)
	}
	return strings.Join(parts, "\n")
}

func (s *SQLStateStore) listGovernanceSQL(query GovernanceQuery) string {
	parts := []string{s.listGovernanceSelectSQL(query), `ORDER BY changed_at DESC, id ASC`, `LIMIT ?`, `OFFSET ?`}
	return strings.Join(parts, "\n")
}

func (s *SQLStateStore) listGovernanceFilterArgs(query GovernanceQuery) []any {
	args := make([]any, 0, 3)
	if query.PackageName != "" {
		args = append(args, query.PackageName)
	}
	if query.TargetType != "" {
		args = append(args, query.TargetType)
	}
	if query.Version != "" {
		args = append(args, query.Version)
	}
	return args
}

func (s *SQLStateStore) listGovernanceArgs(query GovernanceQuery) []any {
	args := s.listGovernanceFilterArgs(query)
	args = append(args, query.PageSize+1, query.Cursor)
	return args
}

func (s *SQLStateStore) versionOrderBySQL() string {
	if s.dialect == "pgx" {
		return `ORDER BY published_at DESC,
CAST(split_part(version, '.', 1) AS INTEGER) DESC,
CAST(split_part(version, '.', 2) AS INTEGER) DESC,
CAST(split_part(version, '.', 3) AS INTEGER) DESC`
	}
	return `ORDER BY published_at DESC,
CAST(substr(version, 1, instr(version, '.') - 1) AS INTEGER) DESC,
CAST(substr(substr(version, instr(version, '.') + 1), 1, instr(substr(version, instr(version, '.') + 1), '.') - 1) AS INTEGER) DESC,
CAST(substr(substr(version, instr(version, '.') + 1), instr(substr(version, instr(version, '.') + 1), '.') + 1) AS INTEGER) DESC`
}

func normalizedSkillsSort(sortKey string, query string) (string, error) {
	if sortKey == "" {
		if strings.TrimSpace(query) != "" {
			sortKey = "relevance"
		} else {
			sortKey = "updated_at_desc"
		}
	}
	switch sortKey {
	case "relevance", "updated_at_desc", "package_name_asc":
		return sortKey, nil
	default:
		return "", NewError(400, "invalid_request", "sort is invalid.", map[string]any{"sort": sortKey})
	}
}

func skillsOrderBySQL(sortKey string, query string) string {
	switch sortKey {
	case "package_name_asc":
		return `ORDER BY package_name ASC`
	case "updated_at_desc":
		return `ORDER BY updated_at DESC, package_name ASC`
	default:
		if strings.TrimSpace(query) == "" {
			return `ORDER BY updated_at DESC, package_name ASC`
		}
		return `ORDER BY (
	(CASE WHEN LOWER(package_name) LIKE ? THEN 3 ELSE 0 END) +
	(CASE WHEN LOWER(display_name) LIKE ? THEN 2 ELSE 0 END) +
	(CASE WHEN LOWER(description) LIKE ? THEN 1 ELSE 0 END)
) DESC, updated_at DESC, package_name ASC`
	}
}

func validateCursor(cursor int) error {
	if cursor < 0 {
		return NewError(400, "invalid_request", "cursor must be greater than or equal to 0.", map[string]any{"cursor": cursor})
	}
	return nil
}

func allowedPackageFilterSQL(patterns []string) string {
	clauses := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if strings.HasSuffix(pattern, "*") {
			clauses = append(clauses, `package_name LIKE ?`)
			continue
		}
		clauses = append(clauses, `package_name = ?`)
	}
	if len(clauses) == 0 {
		return ""
	}
	return `AND (` + strings.Join(clauses, ` OR `) + `)`
}

func allowedPackageFilterArgs(patterns []string) []any {
	args := make([]any, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if strings.HasSuffix(pattern, "*") {
			args = append(args, strings.TrimSuffix(pattern, "*")+"%")
			continue
		}
		args = append(args, pattern)
	}
	return args
}

func matchesAllowedPackagePatterns(patterns []string, packageName string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if strings.HasSuffix(pattern, "*") {
			if strings.HasPrefix(packageName, strings.TrimSuffix(pattern, "*")) {
				return true
			}
			continue
		}
		if packageName == pattern {
			return true
		}
	}
	return false
}

func (s *SQLStateStore) validateCursorWindow(ctx context.Context, selectSQL string, args []any, cursor int, loaded int) error {
	if cursor == 0 || loaded > 0 {
		return nil
	}

	var count int
	countSQL := `SELECT COUNT(*) FROM (` + selectSQL + `
) AS matched`
	if err := s.db.QueryRowContext(ctx, bindSQL(s.dialect, countSQL), args...).Scan(&count); err != nil {
		return err
	}
	if cursor > count {
		return NewError(400, "invalid_request", "cursor is out of range.", map[string]any{"cursor": cursor})
	}
	return nil
}

func pageInfoForWindow[T any](items []T, cursor int, pageSize int) PageInfo {
	if len(items) <= pageSize {
		return PageInfo{HasNextPage: false}
	}
	nextCursor := formatCursorToken(cursor + pageSize)
	return PageInfo{
		NextCursor:  &nextCursor,
		HasNextPage: true,
	}
}

func pageWindow[T any](items []T, _ int, pageSize int) []T {
	if len(items) > pageSize {
		items = items[:pageSize]
	}
	return append([]T(nil), items...)
}

func scanSkillSummary(scanner interface {
	Scan(dest ...any) error
},
) (SkillSummary, error) {
	var (
		item                SkillSummary
		latestVersion       sql.NullString
		latestVersionStatus sql.NullString
		updatedAtRaw        string
	)
	if err := scanner.Scan(
		&item.PackageName,
		&item.DisplayName,
		&item.Description,
		&item.SourceType,
		&item.Status,
		&latestVersion,
		&latestVersionStatus,
		&updatedAtRaw,
	); err != nil {
		return SkillSummary{}, err
	}
	updatedAt, err := time.Parse(time.RFC3339, updatedAtRaw)
	if err != nil {
		return SkillSummary{}, err
	}
	item.LatestVersion = nullStringPtr(latestVersion)
	item.LatestVersionStatus = nullStringPtr(latestVersionStatus)
	item.UpdatedAt = updatedAt
	return item, nil
}

func scanSkillDetail(scanner interface {
	Scan(dest ...any) error
},
) (SkillDetail, error) {
	var (
		item                SkillDetail
		latestVersion       sql.NullString
		latestVersionStatus sql.NullString
		createdAtRaw        string
		updatedAtRaw        string
	)
	if err := scanner.Scan(
		&item.PackageName,
		&item.DisplayName,
		&item.Description,
		&item.SourceType,
		&item.Status,
		&latestVersion,
		&latestVersionStatus,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return SkillDetail{}, err
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtRaw)
	if err != nil {
		return SkillDetail{}, err
	}
	updatedAt, err := time.Parse(time.RFC3339, updatedAtRaw)
	if err != nil {
		return SkillDetail{}, err
	}
	item.CreatedAt = createdAt
	item.LatestVersion = nullStringPtr(latestVersion)
	item.LatestVersionStatus = nullStringPtr(latestVersionStatus)
	item.UpdatedAt = updatedAt
	return item, nil
}

func scanSkillVersionDetail(scanner interface {
	Scan(dest ...any) error
},
) (SkillVersionDetail, error) {
	var (
		version        SkillVersionDetail
		publishedAtRaw string
		distTagsJSON   string
		componentsJSON string
	)
	if err := scanner.Scan(
		&version.PackageName,
		&version.Version,
		&version.VersionStatus,
		&version.InstallPolicy,
		&publishedAtRaw,
		&version.Registry.NPMRegistry,
		&version.Registry.TarballURL,
		&version.Registry.Integrity,
		&version.Registry.Shasum,
		&distTagsJSON,
		&version.Bundle.SkillManifestDigest,
		&componentsJSON,
	); err != nil {
		return SkillVersionDetail{}, err
	}
	publishedAt, err := time.Parse(time.RFC3339Nano, publishedAtRaw)
	if err != nil {
		return SkillVersionDetail{}, err
	}
	version.PublishedAt = publishedAt
	version.Registry.PublishedAt = publishedAt
	if err := json.Unmarshal([]byte(distTagsJSON), &version.Registry.DistTags); err != nil {
		return SkillVersionDetail{}, err
	}
	if err := json.Unmarshal([]byte(componentsJSON), &version.Bundle.Components); err != nil {
		return SkillVersionDetail{}, err
	}
	return version, nil
}

func scanGovernanceRecord(scanner interface {
	Scan(dest ...any) error
},
) (GovernanceRecord, error) {
	var (
		record                    GovernanceRecord
		version                   sql.NullString
		replacementRecommendation sql.NullString
		changedAtRaw              string
	)
	if err := scanner.Scan(
		&record.ID,
		&record.TargetType,
		&record.PackageName,
		&version,
		&record.Action,
		&record.Reason,
		&replacementRecommendation,
		&record.ChangedBy,
		&changedAtRaw,
	); err != nil {
		return GovernanceRecord{}, err
	}
	changedAt, err := time.Parse(time.RFC3339, changedAtRaw)
	if err != nil {
		return GovernanceRecord{}, err
	}
	record.Version = nullStringPtr(version)
	record.ReplacementRecommendation = nullStringPtr(replacementRecommendation)
	record.ChangedAt = changedAt
	return record, nil
}

func bindSQL(dialect string, query string) string {
	if dialect != "pgx" {
		return query
	}
	var builder strings.Builder
	index := 1
	for _, char := range query {
		if char == '?' {
			builder.WriteString("$")
			builder.WriteString(strconv.Itoa(index))
			index++
			continue
		}
		builder.WriteRune(char)
	}
	return builder.String()
}

func BindSQL(dialect string, query string) string {
	return bindSQL(dialect, query)
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func isIgnorableCatalogAddColumnError(err error) bool {
	if err == nil {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate column") || strings.Contains(message, "already exists")
}

func maxUint64(left uint64, right uint64) uint64 {
	if left > right {
		return left
	}
	return right
}

func cloneAggregate(aggregate AgentSkillAggregate) AgentSkillAggregate {
	return AgentSkillAggregate{
		Revision: aggregate.Revision,
		Skill: SkillSnapshot{
			Detail:         aggregate.Skill.Detail,
			CurationReason: aggregate.Skill.CurationReason,
			Versions:       append([]SkillVersionDetail(nil), aggregate.Skill.Versions...),
		},
		Governance: append([]GovernanceRecord(nil), aggregate.Governance...),
	}
}
