package persistence

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

var ErrIdempotencyRecordNotFound = errors.New("idempotency record not found")

type IdempotencyStore interface {
	Lookup(ctx context.Context, endpoint string, role string, key string) (IdempotencyRecord, error)
	Save(ctx context.Context, endpoint string, role string, key string, record IdempotencyRecord) (bool, error)
	DeleteExpired(ctx context.Context) error
}

type SQLIdempotencyStore struct {
	db      *sql.DB
	dialect string
	ttl     time.Duration
}

type IdempotencyRecord struct {
	RequestHash  string
	Status       int
	ResponseBody []byte
	CreatedAt    time.Time
}

func NewSQLIdempotencyStore(ctx context.Context, databaseURL string, ttl time.Duration) (*SQLIdempotencyStore, error) {
	dialect := sqlDriverNameForURL(databaseURL)
	db, err := sql.Open(dialect, databaseURL)
	if err != nil {
		return nil, err
	}
	store := newSQLIdempotencyStore(db, dialect, ttl)
	if err := store.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func NewSQLIdempotencyStoreWithDB(db *sql.DB, dialect string, ttl time.Duration) *SQLIdempotencyStore {
	return newSQLIdempotencyStore(db, dialect, ttl)
}

func newSQLIdempotencyStore(db *sql.DB, dialect string, ttl time.Duration) *SQLIdempotencyStore {
	return &SQLIdempotencyStore{
		db:      db,
		dialect: dialect,
		ttl:     ttl,
	}
}

func (s *SQLIdempotencyStore) Migrate(ctx context.Context) error {
	for _, statement := range []string{
		`CREATE TABLE IF NOT EXISTS packaging_idempotency_records (
			endpoint TEXT NOT NULL,
			role TEXT NOT NULL,
			idempotency_key TEXT NOT NULL,
			request_hash TEXT NOT NULL,
			status INTEGER NOT NULL,
			response_body TEXT NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (endpoint, role, idempotency_key)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_packaging_idempotency_created_at ON packaging_idempotency_records(created_at)`,
	} {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLIdempotencyStore) Lookup(ctx context.Context, endpoint string, role string, key string) (IdempotencyRecord, error) {
	record := IdempotencyRecord{}
	query := bindSQL(s.dialect, `
SELECT request_hash, status, response_body, created_at
FROM packaging_idempotency_records
WHERE endpoint = ? AND role = ? AND idempotency_key = ?`)
	var createdAtRaw string
	err := s.db.QueryRowContext(ctx, query, endpoint, role, key).Scan(
		&record.RequestHash,
		&record.Status,
		&record.ResponseBody,
		&createdAtRaw,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return IdempotencyRecord{}, ErrIdempotencyRecordNotFound
	}
	if err != nil {
		return IdempotencyRecord{}, err
	}
	createdAt, err := time.Parse(time.RFC3339, createdAtRaw)
	if err != nil {
		return IdempotencyRecord{}, err
	}
	record.CreatedAt = createdAt
	if s.ttl > 0 && createdAt.Before(time.Now().UTC().Add(-s.ttl)) {
		return IdempotencyRecord{}, ErrIdempotencyRecordNotFound
	}
	return record, nil
}

func (s *SQLIdempotencyStore) Save(ctx context.Context, endpoint string, role string, key string, record IdempotencyRecord) (bool, error) {
	query := bindSQL(s.dialect, `
INSERT INTO packaging_idempotency_records (
	endpoint,
	role,
	idempotency_key,
	request_hash,
	status,
	response_body,
	created_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(endpoint, role, idempotency_key) DO NOTHING`)
	result, err := s.db.ExecContext(
		ctx,
		query,
		endpoint,
		role,
		key,
		record.RequestHash,
		record.Status,
		string(record.ResponseBody),
		record.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (s *SQLIdempotencyStore) DeleteExpired(ctx context.Context) error {
	if s.ttl <= 0 {
		return nil
	}
	query := bindSQL(s.dialect, `
DELETE FROM packaging_idempotency_records
WHERE created_at < ?`)
	_, err := s.db.ExecContext(ctx, query, time.Now().UTC().Add(-s.ttl).Format(time.RFC3339))
	return err
}

func (s *SQLIdempotencyStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func sqlDriverNameForURL(databaseURL string) string {
	switch {
	case strings.HasPrefix(databaseURL, "postgres://"), strings.HasPrefix(databaseURL, "postgresql://"):
		return "pgx"
	default:
		return "sqlite"
	}
}

func SQLDriverNameForURL(databaseURL string) string {
	return sqlDriverNameForURL(databaseURL)
}

func bindSQL(dialect string, query string) string {
	if dialect != "pgx" {
		return query
	}
	var builder strings.Builder
	index := 1
	for _, r := range query {
		if r == '?' {
			builder.WriteString("$")
			builder.WriteString(strconv.Itoa(index))
			index++
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func BindSQL(dialect string, query string) string {
	return bindSQL(dialect, query)
}
