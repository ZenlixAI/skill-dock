package persistence

import (
	"reflect"
	"strings"
	"testing"
)

func TestSQLAuditStoreBuildListSQLIncludesOnlyActiveFilters(t *testing.T) {
	store := newSQLAuditStore(nil, "sqlite")
	query := AuditQuery{
		Actor:       "svc-publisher",
		PackageName: "@platform.skill/pdf-skill",
		PageSize:    1,
		Cursor:      3,
	}

	sqlText := store.listSQL(query)
	if strings.Contains(sqlText, "(? = '' OR") {
		t.Fatalf("expected dynamic filters instead of optional predicate placeholders, got %q", sqlText)
	}
	if !strings.Contains(sqlText, "actor = ?") || !strings.Contains(sqlText, "package_name = ?") {
		t.Fatalf("expected active filters in SQL, got %q", sqlText)
	}
	if !strings.Contains(sqlText, "LIMIT ?") || !strings.Contains(sqlText, "OFFSET ?") {
		t.Fatalf("expected SQL pagination clauses, got %q", sqlText)
	}

	args := store.listArgs(query)
	want := []any{"svc-publisher", "@platform.skill/pdf-skill", 2, 3}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("expected dynamic audit args %v, got %v", want, args)
	}
}
