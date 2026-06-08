package persist

import (
	"strings"
	"testing"
)

func TestWorkspaceMigrationStatementsUsePostgresTypes(t *testing.T) {
	statements := workspaceMigrationStatements("pgx")
	if len(statements) == 0 {
		t.Fatal("expected migration statements")
	}

	createTable := statements[0]
	if strings.Contains(createTable, "BLOB") {
		t.Fatalf("expected postgres migration to avoid BLOB, got %q", createTable)
	}
	if !strings.Contains(createTable, "workspace_snapshot BYTEA NOT NULL") {
		t.Fatalf("expected workspace_snapshot BYTEA, got %q", createTable)
	}
	if !strings.Contains(createTable, "bundle_content BYTEA") {
		t.Fatalf("expected bundle_content BYTEA, got %q", createTable)
	}
}

func TestWorkspaceMigrationStatementsUseSQLiteTypes(t *testing.T) {
	statements := workspaceMigrationStatements("sqlite")
	if len(statements) == 0 {
		t.Fatal("expected migration statements")
	}

	createTable := statements[0]
	if !strings.Contains(createTable, "workspace_snapshot BLOB NOT NULL") {
		t.Fatalf("expected sqlite migration to use BLOB, got %q", createTable)
	}
	if !strings.Contains(createTable, "bundle_content BLOB") {
		t.Fatalf("expected sqlite migration to use BLOB, got %q", createTable)
	}
}
