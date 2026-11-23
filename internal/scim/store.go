package scim

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"skill_registry/internal/identity"
)

var ErrNotFound = errors.New("scim resource not found")

type User struct {
	ID          string
	ExternalID  string
	UserName    string
	Email       string
	DisplayName string
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Member struct {
	Value   string
	Display string
}

type Group struct {
	ID          string
	ExternalID  string
	DisplayName string
	Members     []Member
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Store struct {
	db       *sql.DB
	dialect  string
	identity identity.Store
}

func NewStore(db *sql.DB, driverName string, identityStore identity.Store) *Store {
	dialect := "sqlite"
	if strings.Contains(strings.ToLower(driverName), "pgx") || strings.Contains(strings.ToLower(driverName), "postgres") {
		dialect = "postgres"
	}
	return &Store{db: db, dialect: dialect, identity: identityStore}
}

func (s *Store) ListUsers(ctx context.Context, filter string, startIndex int, count int) ([]User, int, error) {
	if count <= 0 || count > 500 {
		count = 100
	}
	if startIndex <= 0 {
		startIndex = 1
	}
	query := `SELECT id, external_id, user_name, email, display_name, active, created_at, updated_at FROM scim_users`
	args := []any{}
	if parsed := parseUserFilter(filter); parsed != "" {
		query += ` WHERE lower(user_name) = lower(?) OR lower(email) = lower(?)`
		args = append(args, parsed, parsed)
	}
	query += ` ORDER BY user_name ASC LIMIT ? OFFSET ?`
	args = append(args, count, startIndex-1)
	rows, err := s.db.QueryContext(ctx, s.bind(query), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	countQuery := `SELECT COUNT(1) FROM scim_users`
	if parsed := parseUserFilter(filter); parsed != "" {
		countQuery += ` WHERE lower(user_name) = lower(?) OR lower(email) = lower(?)`
		if err := s.db.QueryRowContext(ctx, s.bind(countQuery), parsed, parsed).Scan(&total); err != nil {
			return nil, 0, err
		}
	} else if err := s.db.QueryRowContext(ctx, s.bind(countQuery)).Scan(&total); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (s *Store) GetUser(ctx context.Context, id string) (User, error) {
	query := s.bind(`SELECT id, external_id, user_name, email, display_name, active, created_at, updated_at FROM scim_users WHERE id = ?`)
	return scanUser(s.db.QueryRowContext(ctx, query, id))
}

func (s *Store) CreateUser(ctx context.Context, user User) (User, error) {
	if strings.TrimSpace(user.UserName) == "" {
		return User{}, errors.New("userName is required")
	}
	if user.ExternalID == "" {
		user.ExternalID = user.UserName
	}
	if s.identity != nil {
		if user.Email != "" {
			if existing, err := s.identity.GetUserByEmail(ctx, user.Email); err == nil {
				user.ID = existing.ID
			}
		}
		if user.ID == "" {
			local, err := s.identity.UpsertUser(ctx, identity.User{
				Issuer:      "scim",
				Subject:     user.ExternalID,
				Email:       user.Email,
				DisplayName: firstNonEmpty(user.DisplayName, user.UserName),
				Status:      "active",
			})
			if err != nil {
				return User{}, err
			}
			user.ID = local.ID
		}
	}
	if user.ID == "" {
		user.ID = stable("scimusr", user.UserName)
	}
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now
	query := s.bind(`INSERT INTO scim_users (id, external_id, user_name, email, display_name, active, raw_json, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?)
ON CONFLICT (id) DO UPDATE SET
  external_id = excluded.external_id,
  user_name = excluded.user_name,
  email = excluded.email,
  display_name = excluded.display_name,
  active = excluded.active,
  updated_at = excluded.updated_at`)
	if _, err := s.db.ExecContext(ctx, query, user.ID, user.ExternalID, user.UserName, user.Email, user.DisplayName, boolInt(user.Active), "{}", now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
		return User{}, err
	}
	return s.GetUser(ctx, user.ID)
}

func (s *Store) ReplaceUser(ctx context.Context, user User) (User, error) {
	if user.ID == "" {
		return User{}, errors.New("id is required")
	}
	query := s.bind(`UPDATE scim_users SET external_id = ?, user_name = ?, email = ?, display_name = ?, active = ?, updated_at = ? WHERE id = ?`)
	if _, err := s.db.ExecContext(ctx, query, user.ExternalID, user.UserName, user.Email, user.DisplayName, boolInt(user.Active), time.Now().UTC().Format(time.RFC3339Nano), user.ID); err != nil {
		return User{}, err
	}
	return s.GetUser(ctx, user.ID)
}

func (s *Store) DeleteUser(ctx context.Context, id string) error {
	query := s.bind(`DELETE FROM scim_users WHERE id = ?`)
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func (s *Store) ListGroups(ctx context.Context, startIndex int, count int) ([]Group, int, error) {
	if count <= 0 || count > 500 {
		count = 100
	}
	if startIndex <= 0 {
		startIndex = 1
	}
	rows, err := s.db.QueryContext(ctx, s.bind(`SELECT id, external_id, display_name, created_at, updated_at FROM scim_groups ORDER BY display_name ASC LIMIT ? OFFSET ?`), count, startIndex-1)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var groups []Group
	for rows.Next() {
		group, err := scanGroup(rows)
		if err != nil {
			return nil, 0, err
		}
		groups = append(groups, group)
	}
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM scim_groups`).Scan(&total); err != nil {
		return nil, 0, err
	}
	return groups, total, rows.Err()
}

func (s *Store) GetGroup(ctx context.Context, id string) (Group, error) {
	query := s.bind(`SELECT id, external_id, display_name, created_at, updated_at FROM scim_groups WHERE id = ?`)
	group, err := scanGroup(s.db.QueryRowContext(ctx, query, id))
	if err != nil {
		return Group{}, err
	}
	members, err := s.groupMembers(ctx, id)
	if err != nil {
		return Group{}, err
	}
	group.Members = members
	return group, nil
}

func (s *Store) CreateGroup(ctx context.Context, group Group) (Group, error) {
	if strings.TrimSpace(group.DisplayName) == "" {
		return Group{}, errors.New("displayName is required")
	}
	if group.ID == "" {
		group.ID = stable("scimgrp", firstNonEmpty(group.ExternalID, group.DisplayName))
	}
	organizationID, err := s.ensureOrganization(ctx, group)
	if err != nil {
		return Group{}, err
	}
	now := time.Now().UTC()
	query := s.bind(`INSERT INTO scim_groups (id, external_id, display_name, organization_id, raw_json, created_at, updated_at)
VALUES (?,?,?,?,?,?,?)
ON CONFLICT (id) DO UPDATE SET
  external_id = excluded.external_id,
  display_name = excluded.display_name,
  organization_id = excluded.organization_id,
  updated_at = excluded.updated_at`)
	if _, err := s.db.ExecContext(ctx, query, group.ID, group.ExternalID, group.DisplayName, organizationID, "{}", now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
		return Group{}, err
	}
	if err := s.replaceGroupMembers(ctx, group); err != nil {
		return Group{}, err
	}
	return s.GetGroup(ctx, group.ID)
}

func (s *Store) ReplaceGroup(ctx context.Context, group Group) (Group, error) {
	if group.ID == "" {
		return Group{}, errors.New("id is required")
	}
	if _, err := s.GetGroup(ctx, group.ID); err != nil {
		return Group{}, err
	}
	if err := s.replaceGroupMembers(ctx, group); err != nil {
		return Group{}, err
	}
	query := s.bind(`UPDATE scim_groups SET external_id = ?, display_name = ?, updated_at = ? WHERE id = ?`)
	if _, err := s.db.ExecContext(ctx, query, group.ExternalID, group.DisplayName, time.Now().UTC().Format(time.RFC3339Nano), group.ID); err != nil {
		return Group{}, err
	}
	return s.GetGroup(ctx, group.ID)
}

func (s *Store) DeleteGroup(ctx context.Context, id string) error {
	query := s.bind(`DELETE FROM scim_groups WHERE id = ?`)
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func (s *Store) ensureOrganization(ctx context.Context, group Group) (string, error) {
	if s.identity == nil {
		return "", errors.New("identity store is not configured")
	}
	slug := normalizeSlug(firstNonEmpty(group.ExternalID, group.DisplayName))
	if org, err := s.identity.GetOrganizationBySlug(ctx, slug); err == nil {
		return org.ID, nil
	}
	logtoOrgID := "scim:" + firstNonEmpty(group.ExternalID, group.ID)
	if org, err := s.identity.GetOrganizationByLogtoID(ctx, logtoOrgID); err == nil {
		return org.ID, nil
	}
	org, err := s.identity.UpsertOrganization(ctx, identity.Organization{
		LogtoOrgID: logtoOrgID,
		Slug:       slug,
		Name:       group.DisplayName,
		Status:     "active",
	})
	if err != nil {
		return "", err
	}
	return org.ID, nil
}

func (s *Store) replaceGroupMembers(ctx context.Context, group Group) error {
	if s.identity == nil {
		return nil
	}
	orgID, err := s.groupOrganizationID(ctx, group.ID)
	if err != nil {
		return err
	}
	for _, member := range group.Members {
		user, err := s.GetUser(ctx, member.Value)
		if err != nil {
			continue
		}
		if err := s.identity.UpsertMembership(ctx, identity.Membership{OrganizationID: orgID, UserID: user.ID}); err != nil {
			return err
		}
		query := s.bind(`INSERT INTO scim_group_members (group_id, user_id) VALUES (?,?) ON CONFLICT (group_id, user_id) DO NOTHING`)
		_, _ = s.db.ExecContext(ctx, query, group.ID, member.Value)
	}
	return nil
}

func (s *Store) groupOrganizationID(ctx context.Context, groupID string) (string, error) {
	query := s.bind(`SELECT COALESCE(organization_id,'') FROM scim_groups WHERE id = ?`)
	var orgID string
	if err := s.db.QueryRowContext(ctx, query, groupID).Scan(&orgID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return orgID, nil
}

func (s *Store) groupMembers(ctx context.Context, groupID string) ([]Member, error) {
	query := s.bind(`SELECT u.id, u.user_name, u.display_name FROM scim_group_members m JOIN scim_users u ON u.id = m.user_id WHERE m.group_id = ? ORDER BY u.user_name ASC`)
	rows, err := s.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []Member
	for rows.Next() {
		var id, userName, displayName string
		if err := rows.Scan(&id, &userName, &displayName); err != nil {
			return nil, err
		}
		members = append(members, Member{Value: id, Display: firstNonEmpty(displayName, userName)})
	}
	return members, rows.Err()
}

func (s *Store) bind(query string) string {
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

func scanUser(row interface{ Scan(...any) error }) (User, error) {
	var (
		user      User
		active    int
		createdAt string
		updatedAt string
	)
	err := row.Scan(&user.ID, &user.ExternalID, &user.UserName, &user.Email, &user.DisplayName, &active, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	user.Active = active != 0
	user.CreatedAt = parseTime(createdAt)
	user.UpdatedAt = parseTime(updatedAt)
	return user, nil
}

func scanGroup(row interface{ Scan(...any) error }) (Group, error) {
	var (
		group     Group
		createdAt string
		updatedAt string
	)
	err := row.Scan(&group.ID, &group.ExternalID, &group.DisplayName, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Group{}, ErrNotFound
	}
	if err != nil {
		return Group{}, err
	}
	group.CreatedAt = parseTime(createdAt)
	group.UpdatedAt = parseTime(updatedAt)
	return group, nil
}

func parseUserFilter(filter string) string {
	filter = strings.TrimSpace(filter)
	lower := strings.ToLower(filter)
	for _, prefix := range []string{`username eq "`, `username eq '`, `emails.value eq "`, `emails.value eq '`} {
		if strings.HasPrefix(lower, prefix) {
			value := filter[len(prefix):]
			value = strings.TrimSuffix(value, `"`)
			value = strings.TrimSuffix(value, `'`)
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z', char >= '0' && char <= '9':
			builder.WriteRune(char)
		case char == '-' || char == '_':
			builder.WriteRune(char)
		default:
			builder.WriteRune('-')
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "scim-tenant"
	}
	return result
}

func stable(prefix string, values ...string) string {
	h := sha256.New()
	for _, value := range values {
		_, _ = h.Write([]byte(value))
		_, _ = h.Write([]byte{0})
	}
	return prefix + "_" + hex.EncodeToString(h.Sum(nil))[:24]
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		parsed, _ = time.Parse(time.RFC3339, value)
	}
	return parsed
}
