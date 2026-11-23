package identity

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
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

func (s *SQLStore) Migrate(ctx context.Context) error {
	return migrate(ctx, s.db, s.dialect)
}

func (s *SQLStore) UpsertUser(ctx context.Context, user User) (User, error) {
	now := time.Now().UTC()
	if user.ID == "" {
		user.ID = stableID("usr", user.Issuer, user.Subject)
	}
	if user.Status == "" {
		user.Status = "active"
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	if s.dialect == "postgres" {
		query := `INSERT INTO users (id, issuer, subject, email, display_name, avatar_url, status, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (issuer, subject) DO UPDATE SET
  email = EXCLUDED.email,
  display_name = EXCLUDED.display_name,
  avatar_url = EXCLUDED.avatar_url,
  status = EXCLUDED.status,
  updated_at = EXCLUDED.updated_at`
		if _, err := s.db.ExecContext(ctx, query, user.ID, user.Issuer, user.Subject, user.Email, user.DisplayName, user.AvatarURL, user.Status, fmtTime(user.CreatedAt), fmtTime(user.UpdatedAt)); err != nil {
			return User{}, err
		}
		return s.GetUserBySubject(ctx, user.Issuer, user.Subject)
	}

	query := `INSERT INTO users (id, issuer, subject, email, display_name, avatar_url, status, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?)
ON CONFLICT (issuer, subject) DO UPDATE SET
  email = excluded.email,
  display_name = excluded.display_name,
  avatar_url = excluded.avatar_url,
  status = excluded.status,
  updated_at = excluded.updated_at`
	if _, err := s.db.ExecContext(ctx, query, user.ID, user.Issuer, user.Subject, user.Email, user.DisplayName, user.AvatarURL, user.Status, fmtTime(user.CreatedAt), fmtTime(user.UpdatedAt)); err != nil {
		return User{}, err
	}
	return s.GetUserBySubject(ctx, user.Issuer, user.Subject)
}

func (s *SQLStore) GetUserByID(ctx context.Context, id string) (User, error) {
	query := s.bind(`SELECT id, issuer, subject, email, display_name, avatar_url, status, created_at, updated_at FROM users WHERE id = ?`)
	return s.scanUser(s.db.QueryRowContext(ctx, query, id))
}

func (s *SQLStore) GetUserBySubject(ctx context.Context, issuer string, subject string) (User, error) {
	query := s.bind(`SELECT id, issuer, subject, email, display_name, avatar_url, status, created_at, updated_at FROM users WHERE issuer = ? AND subject = ?`)
	return s.scanUser(s.db.QueryRowContext(ctx, query, issuer, subject))
}

func (s *SQLStore) GetUserByEmail(ctx context.Context, email string) (User, error) {
	query := s.bind(`SELECT id, issuer, subject, email, display_name, avatar_url, status, created_at, updated_at FROM users WHERE lower(email) = lower(?) LIMIT 1`)
	return s.scanUser(s.db.QueryRowContext(ctx, query, strings.TrimSpace(email)))
}

func (s *SQLStore) UpdateUserIdentity(ctx context.Context, userID string, issuer string, subject string) (User, error) {
	query := s.bind(`UPDATE users SET issuer = ?, subject = ?, updated_at = ? WHERE id = ?`)
	if _, err := s.db.ExecContext(ctx, query, issuer, subject, fmtTime(time.Now().UTC()), userID); err != nil {
		return User{}, err
	}
	return s.GetUserByID(ctx, userID)
}

func (s *SQLStore) ListUsers(ctx context.Context, queryText string, limit int) ([]User, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT id, issuer, subject, email, display_name, avatar_url, status, created_at, updated_at FROM users`
	args := []any{}
	if strings.TrimSpace(queryText) != "" {
		needle := "%" + strings.ToLower(strings.TrimSpace(queryText)) + "%"
		query += ` WHERE lower(email) LIKE ? OR lower(display_name) LIKE ? OR lower(subject) LIKE ?`
		args = append(args, needle, needle, needle)
	}
	query += ` ORDER BY created_at ASC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, s.bind(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		user, scanErr := scanUser(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *SQLStore) UpsertOrganization(ctx context.Context, organization Organization) (Organization, error) {
	now := time.Now().UTC()
	if organization.ID == "" {
		organization.ID = stableID("org", organization.LogtoOrgID)
	}
	if organization.Status == "" {
		organization.Status = "active"
	}
	if organization.Slug == "" {
		organization.Slug = organization.LogtoOrgID
	}
	if organization.Name == "" {
		organization.Name = organization.Slug
	}
	if organization.CreatedAt.IsZero() {
		organization.CreatedAt = now
	}
	organization.UpdatedAt = now
	organization.SyncedAt = now

	query := s.bind(`INSERT INTO organizations (id, logto_org_id, slug, name, status, synced_at, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?)
ON CONFLICT (logto_org_id) DO UPDATE SET
  slug = excluded.slug,
  name = excluded.name,
  status = excluded.status,
  synced_at = excluded.synced_at,
  updated_at = excluded.updated_at`)
	if _, err := s.db.ExecContext(ctx, query, organization.ID, organization.LogtoOrgID, organization.Slug, organization.Name, organization.Status, fmtTime(organization.SyncedAt), fmtTime(organization.CreatedAt), fmtTime(organization.UpdatedAt)); err != nil {
		return Organization{}, err
	}
	return s.GetOrganizationByLogtoID(ctx, organization.LogtoOrgID)
}

func (s *SQLStore) GetOrganizationByID(ctx context.Context, id string) (Organization, error) {
	query := s.bind(`SELECT id, logto_org_id, slug, name, status, synced_at, created_at, updated_at FROM organizations WHERE id = ?`)
	return s.scanOrganization(s.db.QueryRowContext(ctx, query, id))
}

func (s *SQLStore) GetOrganizationByLogtoID(ctx context.Context, logtoOrgID string) (Organization, error) {
	query := s.bind(`SELECT id, logto_org_id, slug, name, status, synced_at, created_at, updated_at FROM organizations WHERE logto_org_id = ?`)
	return s.scanOrganization(s.db.QueryRowContext(ctx, query, logtoOrgID))
}

func (s *SQLStore) GetOrganizationBySlug(ctx context.Context, slug string) (Organization, error) {
	query := s.bind(`SELECT id, logto_org_id, slug, name, status, synced_at, created_at, updated_at FROM organizations WHERE slug = ?`)
	return s.scanOrganization(s.db.QueryRowContext(ctx, query, slug))
}

func (s *SQLStore) ListOrganizationsForUser(ctx context.Context, userID string) ([]OrganizationRole, error) {
	query := s.bind(`SELECT o.id, o.logto_org_id, o.slug, o.name, o.status, o.synced_at, o.created_at, o.updated_at, m.roles_json
FROM organizations o
JOIN organization_members m ON m.organization_id = o.id
WHERE m.user_id = ?
ORDER BY o.slug ASC`)
	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []OrganizationRole
	for rows.Next() {
		var (
			org       Organization
			syncedAt  string
			createdAt string
			updatedAt string
			rolesJSON string
		)
		if err := rows.Scan(&org.ID, &org.LogtoOrgID, &org.Slug, &org.Name, &org.Status, &syncedAt, &createdAt, &updatedAt, &rolesJSON); err != nil {
			return nil, err
		}
		org.SyncedAt = parseTime(syncedAt)
		org.CreatedAt = parseTime(createdAt)
		org.UpdatedAt = parseTime(updatedAt)
		var roles []string
		_ = json.Unmarshal([]byte(rolesJSON), &roles)
		role := ""
		if len(roles) > 0 {
			role = roles[0]
		}
		result = append(result, OrganizationRole{Organization: org, Role: role})
	}
	return result, rows.Err()
}

func (s *SQLStore) UpsertMembership(ctx context.Context, membership Membership) error {
	if strings.TrimSpace(membership.OrganizationID) == "" || strings.TrimSpace(membership.UserID) == "" {
		return errors.New("organization and user are required")
	}
	if membership.Roles == nil {
		membership.Roles = []string{}
	}
	rolesJSON, err := json.Marshal(membership.Roles)
	if err != nil {
		return err
	}
	query := s.bind(`INSERT INTO organization_members (organization_id, user_id, roles_json, synced_at)
VALUES (?,?,?,?)
ON CONFLICT (organization_id, user_id) DO UPDATE SET
  roles_json = CASE WHEN excluded.roles_json = '[]' THEN organization_members.roles_json ELSE excluded.roles_json END,
  synced_at = excluded.synced_at`)
	_, err = s.db.ExecContext(ctx, query, membership.OrganizationID, membership.UserID, string(rolesJSON), fmtTime(time.Now().UTC()))
	return err
}

func (s *SQLStore) RemoveMembership(ctx context.Context, organizationID string, userID string) error {
	query := s.bind(`DELETE FROM organization_members WHERE organization_id = ? AND user_id = ?`)
	_, err := s.db.ExecContext(ctx, query, organizationID, userID)
	return err
}

func (s *SQLStore) IsMember(ctx context.Context, organizationID string, userID string) (bool, error) {
	query := s.bind(`SELECT COUNT(1) FROM organization_members WHERE organization_id = ? AND user_id = ?`)
	var count int
	if err := s.db.QueryRowContext(ctx, query, organizationID, userID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SQLStore) bind(query string) string {
	if s.dialect != "postgres" {
		return query
	}
	var b strings.Builder
	index := 1
	for _, char := range query {
		if char == '?' {
			b.WriteString(fmt.Sprintf("$%d", index))
			index++
			continue
		}
		b.WriteRune(char)
	}
	return b.String()
}

func (s *SQLStore) scanUser(row interface{ Scan(...any) error }) (User, error) {
	var (
		user      User
		createdAt string
		updatedAt string
	)
	if err := row.Scan(&user.ID, &user.Issuer, &user.Subject, &user.Email, &user.DisplayName, &user.AvatarURL, &user.Status, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	user.CreatedAt = parseTime(createdAt)
	user.UpdatedAt = parseTime(updatedAt)
	return user, nil
}

func (s *SQLStore) scanOrganization(row interface{ Scan(...any) error }) (Organization, error) {
	var (
		org       Organization
		syncedAt  string
		createdAt string
		updatedAt string
	)
	if err := row.Scan(&org.ID, &org.LogtoOrgID, &org.Slug, &org.Name, &org.Status, &syncedAt, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Organization{}, ErrNotFound
		}
		return Organization{}, err
	}
	org.SyncedAt = parseTime(syncedAt)
	org.CreatedAt = parseTime(createdAt)
	org.UpdatedAt = parseTime(updatedAt)
	return org, nil
}

func scanUser(row interface{ Scan(...any) error }) (User, error) {
	var (
		user      User
		createdAt string
		updatedAt string
	)
	if err := row.Scan(&user.ID, &user.Issuer, &user.Subject, &user.Email, &user.DisplayName, &user.AvatarURL, &user.Status, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}
	user.CreatedAt = parseTime(createdAt)
	user.UpdatedAt = parseTime(updatedAt)
	return user, nil
}

func stableID(prefix string, parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return prefix + "_" + hex.EncodeToString(h.Sum(nil))[:24]
}

func fmtTime(value time.Time) string {
	if value.IsZero() {
		value = time.Now().UTC()
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		parsed, _ = time.Parse(time.RFC3339, value)
	}
	return parsed
}
