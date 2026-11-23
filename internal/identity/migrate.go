package identity

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const identitySchemaVersion = 1

func migrate(ctx context.Context, db *sql.DB, dialect string) error {
	if db == nil {
		return nil
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return err
	}

	var count int
	if err := db.QueryRowContext(ctx, bindSQL(dialect, `SELECT COUNT(1) FROM schema_migrations WHERE version = ?`), identitySchemaVersion).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return backfillSkillAccess(ctx, db, dialect)
	}

	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			issuer TEXT NOT NULL,
			subject TEXT NOT NULL,
			email TEXT NOT NULL DEFAULT '',
			display_name TEXT NOT NULL DEFAULT '',
			avatar_url TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'active',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE (issuer, subject)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE TABLE IF NOT EXISTS organizations (
			id TEXT PRIMARY KEY,
			logto_org_id TEXT NOT NULL UNIQUE,
			slug TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			synced_at TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_organizations_slug ON organizations(slug)`,
		`CREATE TABLE IF NOT EXISTS organization_members (
			organization_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			roles_json TEXT NOT NULL DEFAULT '[]',
			synced_at TEXT NOT NULL,
			PRIMARY KEY (organization_id, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_org_members_user ON organization_members(user_id)`,
		`CREATE TABLE IF NOT EXISTS skill_access (
			package_name TEXT PRIMARY KEY,
			organization_id TEXT,
			owner_type TEXT NOT NULL DEFAULT 'platform',
			owner_id TEXT,
			visibility TEXT NOT NULL DEFAULT 'private',
			access_revision INTEGER NOT NULL DEFAULT 1,
			legacy_unowned INTEGER NOT NULL DEFAULT 0,
			created_by TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_skill_access_org ON skill_access(organization_id)`,
		`CREATE INDEX IF NOT EXISTS idx_skill_access_visibility ON skill_access(visibility)`,
		`CREATE TABLE IF NOT EXISTS skill_maintainers (
			package_name TEXT NOT NULL,
			organization_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			granted_by TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			PRIMARY KEY (package_name, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS skill_grants (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL,
			package_name TEXT NOT NULL,
			subject_type TEXT NOT NULL,
			subject_id TEXT NOT NULL,
			permission TEXT NOT NULL,
			granted_by TEXT NOT NULL DEFAULT '',
			expires_at TEXT,
			created_at TEXT NOT NULL,
			UNIQUE (organization_id, package_name, subject_type, subject_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_skill_grants_package ON skill_grants(organization_id, package_name)`,
		`CREATE INDEX IF NOT EXISTS idx_skill_grants_subject ON skill_grants(organization_id, subject_type, subject_id)`,
		`CREATE TABLE IF NOT EXISTS package_namespaces (
			id TEXT PRIMARY KEY,
			prefix TEXT NOT NULL UNIQUE,
			organization_id TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			verification TEXT NOT NULL DEFAULT 'admin',
			verified_at TEXT,
			created_by TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_package_namespaces_org ON package_namespaces(organization_id)`,
		`CREATE TABLE IF NOT EXISTS tenant_artifact_configs (
			organization_id TEXT PRIMARY KEY,
			upstream_npm_registry_url TEXT NOT NULL,
			private_scope_prefix TEXT NOT NULL,
			proxy_mount_path TEXT NOT NULL DEFAULT '/repository/skills-npm',
			auth_mode TEXT NOT NULL DEFAULT 'catalog_proxy',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS tags (
			id TEXT PRIMARY KEY,
			slug TEXT NOT NULL,
			label TEXT NOT NULL,
			scope TEXT NOT NULL,
			organization_id TEXT NOT NULL DEFAULT '',
			visibility TEXT NOT NULL DEFAULT 'private',
			created_by TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE (scope, organization_id, slug)
		)`,
		`CREATE TABLE IF NOT EXISTS skill_tags (
			organization_id TEXT NOT NULL DEFAULT '',
			package_name TEXT NOT NULL,
			tag_id TEXT NOT NULL,
			assigned_by TEXT NOT NULL DEFAULT '',
			assigned_at TEXT NOT NULL,
			PRIMARY KEY (organization_id, package_name, tag_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_skill_tags_tag ON skill_tags(tag_id)`,
		`CREATE TABLE IF NOT EXISTS favorites (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			context_type TEXT NOT NULL,
			organization_id TEXT,
			ref_key TEXT NOT NULL,
			ref_type TEXT NOT NULL,
			package_name TEXT,
			external_provider TEXT,
			external_source TEXT,
			external_skill_id TEXT,
			snapshot_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE (user_id, ref_key)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_favorites_user_context ON favorites(user_id, context_type, organization_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS favorite_labels (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			color TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			UNIQUE (user_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS favorite_label_assignments (
			user_id TEXT NOT NULL,
			favorite_id TEXT NOT NULL,
			label_id TEXT NOT NULL,
			PRIMARY KEY (favorite_id, label_id)
		)`,
		`CREATE TABLE IF NOT EXISTS scim_users (
			id TEXT PRIMARY KEY,
			external_id TEXT NOT NULL DEFAULT '',
			user_name TEXT NOT NULL,
			email TEXT NOT NULL DEFAULT '',
			display_name TEXT NOT NULL DEFAULT '',
			active INTEGER NOT NULL DEFAULT 1,
			raw_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE (user_name)
		)`,
		`CREATE TABLE IF NOT EXISTS scim_groups (
			id TEXT PRIMARY KEY,
			external_id TEXT NOT NULL DEFAULT '',
			display_name TEXT NOT NULL,
			organization_id TEXT,
			raw_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS scim_group_members (
			group_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			PRIMARY KEY (group_id, user_id)
		)`,
		`ALTER TABLE governance_records ADD COLUMN changed_by_user_id TEXT`,
		`ALTER TABLE governance_records ADD COLUMN changed_by_organization_id TEXT`,
		`ALTER TABLE governance_records ADD COLUMN changed_by_display TEXT`,
		`ALTER TABLE governance_records ADD COLUMN organization_id TEXT`,
	}

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil && !ignorableMigrationError(err) {
			return fmt.Errorf("identity migration: %w", err)
		}
	}

	if err := backfillSkillAccess(ctx, db, dialect); err != nil {
		return err
	}

	if _, err := db.ExecContext(ctx, bindSQL(dialect, `INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`), identitySchemaVersion, fmtTime(time.Now().UTC())); err != nil {
		return err
	}
	return nil
}

func backfillSkillAccess(ctx context.Context, db *sql.DB, dialect string) error {
	backfill := bindSQL(dialect, `INSERT INTO skill_access (package_name, organization_id, owner_type, owner_id, visibility, access_revision, legacy_unowned, created_by, created_at, updated_at)
SELECT package_name, NULL, 'platform', NULL, 'public', 1, 1, '', created_at, updated_at
FROM skills
WHERE NOT EXISTS (SELECT 1 FROM skill_access WHERE skill_access.package_name = skills.package_name)`)
	if _, err := db.ExecContext(ctx, backfill); err != nil && !ignorableMigrationError(err) {
		return fmt.Errorf("identity migration backfill: %w", err)
	}
	return nil
}

func bindSQL(dialect string, query string) string {
	if dialect != "postgres" {
		return query
	}
	var builder strings.Builder
	index := 1
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

func ignorableMigrationError(err error) bool {
	if err == nil {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate column") ||
		strings.Contains(message, "already exists") ||
		strings.Contains(message, "no such table") ||
		strings.Contains(message, "does not exist")
}
