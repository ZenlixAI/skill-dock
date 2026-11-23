package infra

import (
	"context"
	"database/sql"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"

	"skill_registry/internal/identity"
)

func OpenSQLStateStore(ctx context.Context, databaseURL string) (*sql.DB, string, *SQLStateStore, error) {
	driverName := SQLDriverNameForURL(databaseURL)
	db, err := sql.Open(driverName, databaseURL)
	if err != nil {
		return nil, "", nil, err
	}
	store, err := PrepareSQLStateStore(ctx, db, driverName)
	if err != nil {
		_ = db.Close()
		return nil, "", nil, err
	}
	return db, driverName, store, nil
}

func PrepareSQLStateStore(ctx context.Context, db *sql.DB, driverName string) (*SQLStateStore, error) {
	store := NewSQLStateStore(db, driverName)
	if err := store.Migrate(ctx); err != nil {
		return nil, err
	}
	identityStore := identity.NewSQLStore(db, driverName)
	if err := identityStore.Migrate(ctx); err != nil {
		return nil, err
	}
	return store, nil
}

func SQLDriverNameForURL(databaseURL string) string {
	switch {
	case strings.HasPrefix(databaseURL, "postgres://"), strings.HasPrefix(databaseURL, "postgresql://"):
		return "pgx"
	default:
		return "sqlite"
	}
}
