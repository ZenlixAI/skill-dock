package access

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

type SQLStore struct {
	db      *sql.DB
	dialect string
}

func NewSQLStore(db *sql.DB, driverName string) *SQLStore {
	dialect := "sqlite"
	if strings.Contains(strings.ToLower(driverName), "pgx") || strings.Contains(strings.ToLower(driverName), "postgres") {
		dialect = "postgres"
	}
	return &SQLStore{db: db, dialect: dialect}
}

func (s *SQLStore) GetSkillAccess(ctx context.Context, packageName string) (SkillAccess, error) {
	query := s.bind(`SELECT package_name, COALESCE(organization_id,''), owner_type, COALESCE(owner_id,''), visibility, access_revision, legacy_unowned
FROM skill_access WHERE package_name = ?`)
	var (
		record        SkillAccess
		legacyUnowned int
	)
	err := s.db.QueryRowContext(ctx, query, packageName).Scan(
		&record.PackageName,
		&record.OrganizationID,
		&record.OwnerType,
		&record.OwnerID,
		&record.Visibility,
		&record.AccessRevision,
		&legacyUnowned,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return SkillAccess{}, ErrNotFound
	}
	if err != nil {
		return SkillAccess{}, err
	}
	record.LegacyUnowned = legacyUnowned != 0
	return record, nil
}

func (s *SQLStore) PutSkillAccess(ctx context.Context, record SkillAccess) error {
	if strings.TrimSpace(record.PackageName) == "" {
		return errors.New("package name is required")
	}
	if record.Visibility == "" {
		record.Visibility = "private"
	}
	if record.OwnerType == "" {
		record.OwnerType = "platform"
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	query := s.bind(`INSERT INTO skill_access (package_name, organization_id, owner_type, owner_id, visibility, access_revision, legacy_unowned, created_by, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?)
ON CONFLICT (package_name) DO UPDATE SET
  organization_id = excluded.organization_id,
  owner_type = excluded.owner_type,
  owner_id = excluded.owner_id,
  visibility = excluded.visibility,
  access_revision = skill_access.access_revision + 1,
  legacy_unowned = excluded.legacy_unowned,
  updated_at = excluded.updated_at`)
	_, err := s.db.ExecContext(ctx, query,
		record.PackageName,
		nullable(record.OrganizationID),
		record.OwnerType,
		nullable(record.OwnerID),
		record.Visibility,
		maxInt64(record.AccessRevision, 1),
		boolInt(record.LegacyUnowned),
		"",
		now,
		now,
	)
	return err
}

func (s *SQLStore) DeleteSkillAccess(ctx context.Context, packageName string) error {
	query := s.bind(`DELETE FROM skill_access WHERE package_name = ?`)
	_, err := s.db.ExecContext(ctx, query, packageName)
	return err
}

func (s *SQLStore) ListGrants(ctx context.Context, organizationID string, packageName string) ([]Grant, error) {
	query := s.bind(`SELECT id, organization_id, package_name, subject_type, subject_id, permission, granted_by, expires_at, created_at
FROM skill_grants WHERE organization_id = ? AND package_name = ? ORDER BY created_at ASC`)
	rows, err := s.db.QueryContext(ctx, query, organizationID, packageName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var grants []Grant
	for rows.Next() {
		var (
			grant     Grant
			expiresAt sql.NullString
			createdAt string
		)
		if err := rows.Scan(&grant.ID, &grant.OrganizationID, &grant.PackageName, &grant.SubjectType, &grant.SubjectID, &grant.Permission, &grant.GrantedBy, &expiresAt, &createdAt); err != nil {
			return nil, err
		}
		if expiresAt.Valid && expiresAt.String != "" {
			value := parseTime(expiresAt.String)
			grant.ExpiresAt = &value
		}
		grant.CreatedAt = parseTime(createdAt)
		grants = append(grants, grant)
	}
	return grants, rows.Err()
}

func (s *SQLStore) PutGrant(ctx context.Context, grant Grant) error {
	if grant.ID == "" {
		grant.ID = stable("grt", grant.OrganizationID, grant.PackageName, grant.SubjectType, grant.SubjectID)
	}
	now := time.Now().UTC()
	query := s.bind(`INSERT INTO skill_grants (id, organization_id, package_name, subject_type, subject_id, permission, granted_by, expires_at, created_at)
VALUES (?,?,?,?,?,?,?,?,?)
ON CONFLICT (organization_id, package_name, subject_type, subject_id) DO UPDATE SET
  permission = excluded.permission,
  granted_by = excluded.granted_by,
  expires_at = excluded.expires_at`)
	_, err := s.db.ExecContext(ctx, query,
		grant.ID,
		grant.OrganizationID,
		grant.PackageName,
		grant.SubjectType,
		grant.SubjectID,
		grant.Permission,
		grant.GrantedBy,
		nullableTime(grant.ExpiresAt),
		now.Format(time.RFC3339Nano),
	)
	return err
}

func (s *SQLStore) DeleteGrant(ctx context.Context, organizationID string, packageName string, grantID string) error {
	query := s.bind(`DELETE FROM skill_grants WHERE organization_id = ? AND package_name = ? AND id = ?`)
	_, err := s.db.ExecContext(ctx, query, organizationID, packageName, grantID)
	return err
}

func (s *SQLStore) IsMaintainer(ctx context.Context, packageName string, userID string) (bool, error) {
	query := s.bind(`SELECT COUNT(1) FROM skill_maintainers WHERE package_name = ? AND user_id = ?`)
	var count int
	if err := s.db.QueryRowContext(ctx, query, packageName, userID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SQLStore) ListMaintainers(ctx context.Context, packageName string) ([]string, error) {
	query := s.bind(`SELECT user_id FROM skill_maintainers WHERE package_name = ? ORDER BY created_at ASC`)
	rows, err := s.db.QueryContext(ctx, query, packageName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		users = append(users, userID)
	}
	return users, rows.Err()
}

func (s *SQLStore) AddMaintainer(ctx context.Context, packageName string, organizationID string, userID string, grantedBy string) error {
	query := s.bind(`INSERT INTO skill_maintainers (package_name, organization_id, user_id, granted_by, created_at)
VALUES (?,?,?,?,?) ON CONFLICT (package_name, user_id) DO NOTHING`)
	_, err := s.db.ExecContext(ctx, query, packageName, organizationID, userID, grantedBy, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *SQLStore) RemoveMaintainer(ctx context.Context, packageName string, userID string) error {
	query := s.bind(`DELETE FROM skill_maintainers WHERE package_name = ? AND user_id = ?`)
	_, err := s.db.ExecContext(ctx, query, packageName, userID)
	return err
}

func (s *SQLStore) GetNamespaceForPackage(ctx context.Context, packageName string) (Namespace, error) {
	query := s.bind(`SELECT id, prefix, organization_id, status, verification, verified_at, created_at, updated_at
FROM package_namespaces WHERE ? LIKE prefix || '%' ORDER BY LENGTH(prefix) DESC LIMIT 1`)
	return s.scanNamespace(s.db.QueryRowContext(ctx, query, packageName))
}

func (s *SQLStore) GetNamespaceByPrefix(ctx context.Context, prefix string) (Namespace, error) {
	query := s.bind(`SELECT id, prefix, organization_id, status, verification, verified_at, created_at, updated_at
FROM package_namespaces WHERE prefix = ?`)
	return s.scanNamespace(s.db.QueryRowContext(ctx, query, prefix))
}

func (s *SQLStore) ListNamespaces(ctx context.Context, organizationID string) ([]Namespace, error) {
	query := s.bind(`SELECT id, prefix, organization_id, status, verification, verified_at, created_at, updated_at
FROM package_namespaces WHERE organization_id = ? ORDER BY prefix ASC`)
	rows, err := s.db.QueryContext(ctx, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var namespaces []Namespace
	for rows.Next() {
		namespace, err := s.scanNamespace(rows)
		if err != nil {
			return nil, err
		}
		namespaces = append(namespaces, namespace)
	}
	return namespaces, rows.Err()
}

func (s *SQLStore) PutNamespace(ctx context.Context, namespace Namespace) (Namespace, error) {
	namespace.Prefix = normalizePrefix(namespace.Prefix)
	if namespace.Prefix == "" || namespace.OrganizationID == "" {
		return Namespace{}, errors.New("namespace prefix and organization are required")
	}
	if namespace.ID == "" {
		namespace.ID = stable("ns", namespace.OrganizationID, namespace.Prefix)
	}
	if namespace.Status == "" {
		namespace.Status = "pending"
	}
	if namespace.Verification == "" {
		namespace.Verification = "admin"
	}
	now := time.Now().UTC()
	query := s.bind(`INSERT INTO package_namespaces (id, prefix, organization_id, status, verification, verified_at, created_by, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?)
ON CONFLICT (prefix) DO UPDATE SET
  status = excluded.status,
  verification = excluded.verification,
  verified_at = excluded.verified_at,
  updated_at = excluded.updated_at`)
	_, err := s.db.ExecContext(ctx, query, namespace.ID, namespace.Prefix, namespace.OrganizationID, namespace.Status, namespace.Verification, nullableTime(namespace.VerifiedAt), "", now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return Namespace{}, err
	}
	return s.GetNamespaceByPrefix(ctx, namespace.Prefix)
}

func (s *SQLStore) VerifyNamespace(ctx context.Context, organizationID string, id string) (Namespace, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	query := s.bind(`UPDATE package_namespaces SET status = 'verified', verified_at = ?, updated_at = ? WHERE id = ? AND organization_id = ?`)
	result, err := s.db.ExecContext(ctx, query, now, now, id, organizationID)
	if err != nil {
		return Namespace{}, err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return Namespace{}, ErrNotFound
	}
	query = s.bind(`SELECT id, prefix, organization_id, status, verification, verified_at, created_at, updated_at FROM package_namespaces WHERE id = ?`)
	return s.scanNamespace(s.db.QueryRowContext(ctx, query, id))
}

func (s *SQLStore) GetArtifactConfig(ctx context.Context, organizationID string) (ArtifactConfig, error) {
	query := s.bind(`SELECT organization_id, upstream_npm_registry_url, private_scope_prefix, proxy_mount_path, auth_mode FROM tenant_artifact_configs WHERE organization_id = ?`)
	var config ArtifactConfig
	err := s.db.QueryRowContext(ctx, query, organizationID).Scan(&config.OrganizationID, &config.UpstreamURL, &config.ScopePrefix, &config.ProxyMountPath, &config.AuthMode)
	if errors.Is(err, sql.ErrNoRows) {
		return ArtifactConfig{}, ErrNotFound
	}
	return config, err
}

func (s *SQLStore) PutArtifactConfig(ctx context.Context, config ArtifactConfig) (ArtifactConfig, error) {
	if config.OrganizationID == "" || config.UpstreamURL == "" || config.ScopePrefix == "" {
		return ArtifactConfig{}, errors.New("organization, upstream URL, and scope prefix are required")
	}
	if config.ProxyMountPath == "" {
		config.ProxyMountPath = "/repository/skills-npm"
	}
	if config.AuthMode == "" {
		config.AuthMode = "catalog_proxy"
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	query := s.bind(`INSERT INTO tenant_artifact_configs (organization_id, upstream_npm_registry_url, private_scope_prefix, proxy_mount_path, auth_mode, created_at, updated_at)
VALUES (?,?,?,?,?,?,?)
ON CONFLICT (organization_id) DO UPDATE SET
  upstream_npm_registry_url = excluded.upstream_npm_registry_url,
  private_scope_prefix = excluded.private_scope_prefix,
  proxy_mount_path = excluded.proxy_mount_path,
  auth_mode = excluded.auth_mode,
  updated_at = excluded.updated_at`)
	if _, err := s.db.ExecContext(ctx, query, config.OrganizationID, config.UpstreamURL, config.ScopePrefix, config.ProxyMountPath, config.AuthMode, now, now); err != nil {
		return ArtifactConfig{}, err
	}
	return s.GetArtifactConfig(ctx, config.OrganizationID)
}

func (s *SQLStore) scanNamespace(row interface{ Scan(...any) error }) (Namespace, error) {
	var (
		namespace  Namespace
		verifiedAt sql.NullString
		createdAt  string
		updatedAt  string
	)
	err := row.Scan(&namespace.ID, &namespace.Prefix, &namespace.OrganizationID, &namespace.Status, &namespace.Verification, &verifiedAt, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Namespace{}, ErrNotFound
	}
	if err != nil {
		return Namespace{}, err
	}
	if verifiedAt.Valid && verifiedAt.String != "" {
		value := parseTime(verifiedAt.String)
		namespace.VerifiedAt = &value
	}
	namespace.CreatedAt = parseTime(createdAt)
	namespace.UpdatedAt = parseTime(updatedAt)
	return namespace, nil
}

func (s *SQLStore) bind(query string) string {
	if s.dialect != "postgres" {
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

func normalizePrefix(prefix string) string {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	if prefix == "" {
		return ""
	}
	if !strings.HasPrefix(prefix, "@") {
		prefix = "@" + prefix
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix
}

func stable(prefix string, values ...string) string {
	h := sha256.New()
	for _, value := range values {
		_, _ = h.Write([]byte(value))
		_, _ = h.Write([]byte{0})
	}
	return prefix + "_" + hex.EncodeToString(h.Sum(nil))[:24]
}

func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func maxInt64(value int64, fallback int64) int64 {
	if value > 0 {
		return value
	}
	return fallback
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		parsed, _ = time.Parse(time.RFC3339, value)
	}
	return parsed
}
