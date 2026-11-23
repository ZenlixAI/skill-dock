package favorites

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

var ErrNotFound = errors.New("favorite not found")

type Favorite struct {
	ID               string          `json:"id"`
	UserID           string          `json:"-"`
	ContextType      string          `json:"contextType"`
	OrganizationID   string          `json:"organizationId,omitempty"`
	RefKey           string          `json:"refKey"`
	RefType          string          `json:"refType"`
	PackageName      string          `json:"packageName,omitempty"`
	ExternalProvider string          `json:"externalProvider,omitempty"`
	ExternalSource   string          `json:"externalSource,omitempty"`
	ExternalSkillID  string          `json:"externalSkillId,omitempty"`
	Snapshot         json.RawMessage `json:"snapshot,omitempty"`
	Available        bool            `json:"available"`
	CreatedAt        string          `json:"createdAt"`
	UpdatedAt        string          `json:"updatedAt"`
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

func (s *Store) List(ctx context.Context, userID string, organizationID string, includePersonal bool, limit int, offset int) ([]Favorite, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	query := s.bind(`SELECT id, user_id, context_type, COALESCE(organization_id,''), ref_key, ref_type, COALESCE(package_name,''), COALESCE(external_provider,''), COALESCE(external_source,''), COALESCE(external_skill_id,''), snapshot_json, created_at, updated_at
FROM favorites
WHERE user_id = ?
  AND (context_type = 'global'
    OR (context_type = 'tenant' AND organization_id = ?)
    OR (context_type = 'personal' AND ? = 1))
ORDER BY created_at DESC LIMIT ? OFFSET ?`)
	rows, err := s.db.QueryContext(ctx, query, userID, organizationID, boolInt(includePersonal), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Favorite
	for rows.Next() {
		favorite, err := scanFavorite(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, favorite)
	}
	return result, rows.Err()
}

func (s *Store) Upsert(ctx context.Context, favorite Favorite) (Favorite, error) {
	if favorite.UserID == "" || favorite.RefKey == "" || favorite.RefType == "" {
		return Favorite{}, errors.New("user, refKey, and refType are required")
	}
	if favorite.ID == "" {
		favorite.ID = stable("fav", favorite.UserID, favorite.RefKey)
	}
	if favorite.ContextType == "" {
		favorite.ContextType = "global"
	}
	if favorite.Snapshot == nil {
		favorite.Snapshot = json.RawMessage(`{}`)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	query := s.bind(`INSERT INTO favorites (id, user_id, context_type, organization_id, ref_key, ref_type, package_name, external_provider, external_source, external_skill_id, snapshot_json, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT (user_id, ref_key) DO UPDATE SET
  context_type = excluded.context_type,
  organization_id = excluded.organization_id,
  ref_type = excluded.ref_type,
  package_name = excluded.package_name,
  external_provider = excluded.external_provider,
  external_source = excluded.external_source,
  external_skill_id = excluded.external_skill_id,
  snapshot_json = excluded.snapshot_json,
  updated_at = excluded.updated_at`)
	if _, err := s.db.ExecContext(ctx, query,
		favorite.ID,
		favorite.UserID,
		favorite.ContextType,
		nullable(favorite.OrganizationID),
		favorite.RefKey,
		favorite.RefType,
		nullable(favorite.PackageName),
		nullable(favorite.ExternalProvider),
		nullable(favorite.ExternalSource),
		nullable(favorite.ExternalSkillID),
		string(favorite.Snapshot),
		now,
		now,
	); err != nil {
		return Favorite{}, err
	}
	return s.get(ctx, favorite.UserID, favorite.RefKey)
}

func (s *Store) Delete(ctx context.Context, userID string, refKey string) error {
	query := s.bind(`DELETE FROM favorites WHERE user_id = ? AND ref_key = ?`)
	_, err := s.db.ExecContext(ctx, query, userID, refKey)
	return err
}

func (s *Store) Status(ctx context.Context, userID string, refKeys []string) (map[string]bool, error) {
	result := map[string]bool{}
	if len(refKeys) == 0 {
		return result, nil
	}
	for _, refKey := range refKeys {
		if strings.TrimSpace(refKey) == "" {
			continue
		}
		var count int
		query := s.bind(`SELECT COUNT(1) FROM favorites WHERE user_id = ? AND ref_key = ?`)
		if err := s.db.QueryRowContext(ctx, query, userID, refKey).Scan(&count); err != nil {
			return nil, err
		}
		result[refKey] = count > 0
	}
	return result, nil
}

func (s *Store) get(ctx context.Context, userID string, refKey string) (Favorite, error) {
	query := s.bind(`SELECT id, user_id, context_type, COALESCE(organization_id,''), ref_key, ref_type, COALESCE(package_name,''), COALESCE(external_provider,''), COALESCE(external_source,''), COALESCE(external_skill_id,''), snapshot_json, created_at, updated_at
FROM favorites WHERE user_id = ? AND ref_key = ?`)
	return scanFavorite(s.db.QueryRowContext(ctx, query, userID, refKey))
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

func scanFavorite(row interface{ Scan(...any) error }) (Favorite, error) {
	var (
		favorite     Favorite
		snapshotJSON string
	)
	err := row.Scan(
		&favorite.ID,
		&favorite.UserID,
		&favorite.ContextType,
		&favorite.OrganizationID,
		&favorite.RefKey,
		&favorite.RefType,
		&favorite.PackageName,
		&favorite.ExternalProvider,
		&favorite.ExternalSource,
		&favorite.ExternalSkillID,
		&snapshotJSON,
		&favorite.CreatedAt,
		&favorite.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Favorite{}, ErrNotFound
	}
	if err != nil {
		return Favorite{}, err
	}
	favorite.Snapshot = json.RawMessage(snapshotJSON)
	favorite.Available = true
	return favorite, nil
}

func nullable(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func stable(prefix string, values ...string) string {
	h := sha256.New()
	for _, value := range values {
		_, _ = h.Write([]byte(value))
		_, _ = h.Write([]byte{0})
	}
	return prefix + "_" + hex.EncodeToString(h.Sum(nil))[:24]
}
