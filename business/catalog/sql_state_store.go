package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
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

	if err = s.deleteAggregateRowsTx(ctx, tx, aggregate.Skill.Detail.PackageName); err != nil {
		return err
	}
	if err = s.insertAggregateTx(ctx, tx, cloneAggregate(aggregate), expectedRevision+1); err != nil {
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

	sortKey := query.Sort
	if sortKey == "" {
		if query.Query != "" {
			sortKey = "relevance"
		} else {
			sortKey = "updated_at_desc"
		}
	}
	if sortKey != "relevance" && sortKey != "updated_at_desc" && sortKey != "package_name_asc" {
		return SkillsPage{}, NewError(400, "invalid_request", "sort is invalid.", map[string]any{"sort": sortKey})
	}

	rows, err := s.db.QueryContext(ctx, bindSQL(s.dialect, s.listSkillsSQL(query)), s.listSkillsArgs(query)...)
	if err != nil {
		return SkillsPage{}, err
	}
	defer rows.Close()

	items := make([]SkillSummary, 0)
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

	sortSkills(items, sortKey, query.Query)
	paged, pageInfo, err := paginate(items, query.Cursor, query.PageSize)
	if err != nil {
		return SkillsPage{}, err
	}
	return SkillsPage{Items: paged, PageInfo: pageInfo}, nil
}

func (s *SQLStateStore) ListCuratedSkills(ctx context.Context, pageSize int, cursor int) (CuratedSkillsPage, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		return CuratedSkillsPage{}, NewError(400, "invalid_request", "pageSize must be less than or equal to 100.", map[string]any{"pageSize": pageSize})
	}

	rows, err := s.db.QueryContext(ctx, `
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
WHERE curation_reason <> '' AND status <> 'retired'
ORDER BY updated_at DESC`)
	if err != nil {
		return CuratedSkillsPage{}, err
	}
	defer rows.Close()

	items := make([]CuratedSkillSummary, 0)
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

	paged, pageInfo, err := paginate(items, cursor, pageSize)
	if err != nil {
		return CuratedSkillsPage{}, err
	}
	return CuratedSkillsPage{Items: paged, PageInfo: pageInfo}, nil
}

func (s *SQLStateStore) ListVersions(ctx context.Context, query VersionQuery) (VersionsPage, error) {
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		return VersionsPage{}, NewError(400, "invalid_request", "pageSize must be less than or equal to 100.", map[string]any{"pageSize": query.PageSize})
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

	items := make([]SkillVersionSummary, 0)
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

	paged, pageInfo, err := paginate(items, query.Cursor, query.PageSize)
	if err != nil {
		return VersionsPage{}, err
	}
	return VersionsPage{Items: paged, PageInfo: pageInfo}, nil
}

func (s *SQLStateStore) ListGovernanceChanges(ctx context.Context, query GovernanceQuery) (GovernancePage, error) {
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		return GovernancePage{}, NewError(400, "invalid_request", "pageSize must be less than or equal to 100.", map[string]any{"pageSize": query.PageSize})
	}

	rows, err := s.db.QueryContext(ctx, bindSQL(s.dialect, s.listGovernanceSQL(query)), s.listGovernanceArgs(query)...)
	if err != nil {
		return GovernancePage{}, err
	}
	defer rows.Close()

	items := make([]GovernanceRecord, 0)
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

	paged, pageInfo, err := paginate(items, query.Cursor, query.PageSize)
	if err != nil {
		return GovernancePage{}, err
	}
	return GovernancePage{Items: paged, PageInfo: pageInfo}, nil
}

func (s *SQLStateStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *SQLStateStore) loadVersionsByPackage(ctx context.Context, packageName string) ([]SkillVersionDetail, error) {
	rows, err := s.db.QueryContext(ctx, bindSQL(s.dialect, `
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
ORDER BY published_at DESC`), packageName)
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

func (s *SQLStateStore) deleteAggregateRowsTx(ctx context.Context, tx *sql.Tx, packageName string) error {
	for _, statement := range []string{
		`DELETE FROM governance_records WHERE package_name = ?`,
		`DELETE FROM skill_versions WHERE package_name = ?`,
		`DELETE FROM skills WHERE package_name = ?`,
	} {
		if _, err := tx.ExecContext(ctx, bindSQL(s.dialect, statement), packageName); err != nil {
			return err
		}
	}
	return nil
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

	var latestVersion any
	if aggregate.Skill.Detail.LatestVersion != nil {
		latestVersion = *aggregate.Skill.Detail.LatestVersion
	}
	var latestVersionStatus any
	if aggregate.Skill.Detail.LatestVersionStatus != nil {
		latestVersionStatus = *aggregate.Skill.Detail.LatestVersionStatus
	}

	if _, err := tx.ExecContext(
		ctx,
		insertSkillSQL,
		aggregate.Skill.Detail.PackageName,
		revision,
		aggregate.Skill.Detail.DisplayName,
		aggregate.Skill.Detail.Description,
		string(aggregate.Skill.Detail.SourceType),
		string(aggregate.Skill.Detail.Status),
		latestVersion,
		latestVersionStatus,
		aggregate.Skill.Detail.CreatedAt.UTC().Format(time.RFC3339),
		aggregate.Skill.Detail.UpdatedAt.UTC().Format(time.RFC3339),
		aggregate.Skill.CurationReason,
	); err != nil {
		return err
	}

	for _, version := range aggregate.Skill.Versions {
		distTagsJSON, err := json.Marshal(version.Registry.DistTags)
		if err != nil {
			return err
		}
		componentsJSON, err := json.Marshal(version.Bundle.Components)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			insertVersionSQL,
			version.PackageName,
			version.Version,
			string(version.VersionStatus),
			string(version.InstallPolicy),
			version.PublishedAt.UTC().Format(time.RFC3339),
			version.Registry.NPMRegistry,
			version.Registry.TarballURL,
			version.Registry.Integrity,
			version.Registry.Shasum,
			string(distTagsJSON),
			version.Bundle.SkillManifestDigest,
			string(componentsJSON),
		); err != nil {
			return err
		}
	}

	for _, record := range aggregate.Governance {
		var version any
		if record.Version != nil {
			version = *record.Version
		}
		var replacementRecommendation any
		if record.ReplacementRecommendation != nil {
			replacementRecommendation = *record.ReplacementRecommendation
		}
		if _, err := tx.ExecContext(
			ctx,
			insertGovernanceSQL,
			record.ID,
			string(record.TargetType),
			record.PackageName,
			version,
			string(record.Action),
			record.Reason,
			replacementRecommendation,
			record.ChangedBy,
			record.ChangedAt.UTC().Format(time.RFC3339),
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLStateStore) skillExists(ctx context.Context, packageName string) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, bindSQL(s.dialect, `SELECT COUNT(*) FROM skills WHERE package_name = ?`), packageName).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SQLStateStore) listSkillsSQL(query SkillQuery) string {
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
	if strings.TrimSpace(query.Query) != "" {
		parts = append(parts, `AND (LOWER(package_name) LIKE ? OR LOWER(display_name) LIKE ? OR LOWER(description) LIKE ?)`)
	}
	return strings.Join(parts, "\n")
}

func (s *SQLStateStore) listSkillsArgs(query SkillQuery) []any {
	args := make([]any, 0, 5)
	if query.PackageName != "" {
		args = append(args, query.PackageName)
	}
	if query.SourceType != "" {
		args = append(args, query.SourceType)
	}
	if strings.TrimSpace(query.Query) != "" {
		needle := "%" + strings.ToLower(strings.TrimSpace(query.Query)) + "%"
		args = append(args, needle, needle, needle)
	}
	return args
}

func (s *SQLStateStore) listVersionsSQL(query VersionQuery) string {
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
	parts = append(parts, `ORDER BY published_at DESC`)
	return strings.Join(parts, "\n")
}

func (s *SQLStateStore) listVersionsArgs(query VersionQuery) []any {
	return []any{query.PackageName}
}

func (s *SQLStateStore) listGovernanceSQL(query GovernanceQuery) string {
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
	parts = append(parts, `ORDER BY changed_at DESC, id ASC`)
	return strings.Join(parts, "\n")
}

func (s *SQLStateStore) listGovernanceArgs(query GovernanceQuery) []any {
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

func scanSkillSummary(scanner interface {
	Scan(dest ...any) error
}) (SkillSummary, error) {
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

func scanSkillVersionDetail(scanner interface {
	Scan(dest ...any) error
}) (SkillVersionDetail, error) {
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
	publishedAt, err := time.Parse(time.RFC3339, publishedAtRaw)
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
}) (GovernanceRecord, error) {
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
