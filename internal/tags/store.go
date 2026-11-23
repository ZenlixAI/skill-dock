package tags

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

var ErrNotFound = errors.New("tag not found")

type Tag struct {
	ID             string `json:"id"`
	Slug           string `json:"slug"`
	Label          string `json:"label"`
	Scope          string `json:"scope"`
	OrganizationID string `json:"organizationId,omitempty"`
	Visibility     string `json:"visibility"`
	CreatedBy      string `json:"createdBy,omitempty"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

type Store struct {
	db      *sql.DB
	dialect string
}

func NewStore(db *sql.DB, driverName string) *Store {
	dialect := "sqlite"
	if strings.Contains(strings.ToLower(driverName), "pgx") || strings.Contains(strings.ToLower(driverName), "postgres") {
		dialect = "postgres"
	}
	return &Store{db: db, dialect: dialect}
}

func (s *Store) ListVisible(ctx context.Context, organizationID string) ([]Tag, error) {
	query := s.bind(`SELECT id, slug, label, scope, organization_id, visibility, created_by, created_at, updated_at
FROM tags WHERE scope = 'system' OR (scope = 'tenant' AND organization_id = ?)
ORDER BY scope ASC, slug ASC`)
	rows, err := s.db.QueryContext(ctx, query, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Tag
	for rows.Next() {
		tag, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, tag)
	}
	return result, rows.Err()
}

func (s *Store) ListForPackage(ctx context.Context, packageName string) ([]Tag, error) {
	query := s.bind(`SELECT t.id, t.slug, t.label, t.scope, t.organization_id, t.visibility, t.created_by, t.created_at, t.updated_at
FROM tags t JOIN skill_tags st ON st.tag_id = t.id
WHERE st.package_name = ? ORDER BY t.scope ASC, t.slug ASC`)
	rows, err := s.db.QueryContext(ctx, query, packageName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Tag
	for rows.Next() {
		tag, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, tag)
	}
	return result, rows.Err()
}

func (s *Store) Get(ctx context.Context, id string) (Tag, error) {
	return s.get(ctx, id)
}

func (s *Store) ListForPackages(ctx context.Context, packageNames []string) (map[string][]Tag, error) {
	result := map[string][]Tag{}
	if len(packageNames) == 0 {
		return result, nil
	}
	placeholders := make([]string, 0, len(packageNames))
	args := make([]any, 0, len(packageNames))
	for _, packageName := range packageNames {
		if strings.TrimSpace(packageName) == "" {
			continue
		}
		placeholders = append(placeholders, "?")
		args = append(args, packageName)
	}
	if len(args) == 0 {
		return result, nil
	}
	query := s.bind(`SELECT st.package_name, t.id, t.slug, t.label, t.scope, t.organization_id, t.visibility, t.created_by, t.created_at, t.updated_at
FROM tags t JOIN skill_tags st ON st.tag_id = t.id
WHERE st.package_name IN (` + strings.Join(placeholders, ",") + `)
ORDER BY t.scope ASC, t.slug ASC`)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var packageName string
		var tag Tag
		if err := rows.Scan(&packageName, &tag.ID, &tag.Slug, &tag.Label, &tag.Scope, &tag.OrganizationID, &tag.Visibility, &tag.CreatedBy, &tag.CreatedAt, &tag.UpdatedAt); err != nil {
			return nil, err
		}
		result[packageName] = append(result[packageName], tag)
	}
	return result, rows.Err()
}

func (s *Store) Create(ctx context.Context, tag Tag) (Tag, error) {
	tag.Slug = normalizeSlug(tag.Slug)
	if tag.Slug == "" || tag.Label == "" || tag.Scope == "" {
		return Tag{}, errors.New("slug, label, and scope are required")
	}
	if tag.Scope != "system" && tag.Scope != "tenant" {
		return Tag{}, errors.New("scope must be system or tenant")
	}
	if tag.Scope == "tenant" && tag.OrganizationID == "" {
		return Tag{}, errors.New("tenant tag requires organization")
	}
	if tag.Visibility == "" {
		tag.Visibility = "private"
	}
	if tag.ID == "" {
		tag.ID = stable("tag", tag.Scope, tag.OrganizationID, tag.Slug)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	query := s.bind(`INSERT INTO tags (id, slug, label, scope, organization_id, visibility, created_by, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?)
ON CONFLICT (scope, organization_id, slug) DO UPDATE SET
  label = excluded.label,
  visibility = excluded.visibility,
  updated_at = excluded.updated_at`)
	if _, err := s.db.ExecContext(ctx, query, tag.ID, tag.Slug, tag.Label, tag.Scope, tag.OrganizationID, tag.Visibility, tag.CreatedBy, now, now); err != nil {
		return Tag{}, err
	}
	return s.get(ctx, tag.ID)
}

func (s *Store) Update(ctx context.Context, tag Tag) (Tag, error) {
	query := s.bind(`UPDATE tags SET label = ?, visibility = ?, updated_at = ? WHERE id = ?`)
	if _, err := s.db.ExecContext(ctx, query, tag.Label, tag.Visibility, time.Now().UTC().Format(time.RFC3339Nano), tag.ID); err != nil {
		return Tag{}, err
	}
	return s.get(ctx, tag.ID)
}

func (s *Store) Delete(ctx context.Context, id string) error {
	query := s.bind(`DELETE FROM tags WHERE id = ?`)
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func (s *Store) ReplaceSkillTags(ctx context.Context, organizationID string, packageName string, tagIDs []string, assignedBy string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, s.bind(`DELETE FROM skill_tags WHERE organization_id = ? AND package_name = ?`), organizationID, packageName); err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		if strings.TrimSpace(tagID) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, s.bind(`INSERT INTO skill_tags (organization_id, package_name, tag_id, assigned_by, assigned_at) VALUES (?,?,?,?,?)`), organizationID, packageName, tagID, assignedBy, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) get(ctx context.Context, id string) (Tag, error) {
	query := s.bind(`SELECT id, slug, label, scope, organization_id, visibility, created_by, created_at, updated_at FROM tags WHERE id = ?`)
	return scanTag(s.db.QueryRowContext(ctx, query, id))
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

func scanTag(row interface{ Scan(...any) error }) (Tag, error) {
	var tag Tag
	if err := row.Scan(&tag.ID, &tag.Slug, &tag.Label, &tag.Scope, &tag.OrganizationID, &tag.Visibility, &tag.CreatedBy, &tag.CreatedAt, &tag.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Tag{}, ErrNotFound
		}
		return Tag{}, err
	}
	return tag, nil
}

func normalizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	return strings.Trim(value, "-")
}

func stable(prefix string, values ...string) string {
	h := sha256.New()
	for _, value := range values {
		_, _ = h.Write([]byte(value))
		_, _ = h.Write([]byte{0})
	}
	return prefix + "_" + hex.EncodeToString(h.Sum(nil))[:24]
}
