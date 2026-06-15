package infra

import (
	"reflect"
	"strings"
	"testing"

	. "skill_registry/internal/catalog/domain"
)

func TestSQLStateStoreListSkillsSQLAppliesSortLimitOffset(t *testing.T) {
	store := &SQLStateStore{dialect: "sqlite"}
	query := SkillQuery{
		Sort:     "package_name_asc",
		PageSize: 1,
		Cursor:   1,
	}

	sortKey, err := normalizedSkillsSort(query.Sort, query.Query)
	if err != nil {
		t.Fatalf("normalize sort: %v", err)
	}

	sqlText := store.listSkillsSQL(query, sortKey)
	if !strings.Contains(sqlText, "ORDER BY package_name ASC") {
		t.Fatalf("expected package sort in SQL, got %q", sqlText)
	}
	if !strings.Contains(sqlText, "LIMIT ?") {
		t.Fatalf("expected SQL limit clause, got %q", sqlText)
	}
	if !strings.Contains(sqlText, "OFFSET ?") {
		t.Fatalf("expected SQL offset clause, got %q", sqlText)
	}

	args := store.listSkillsArgs(query, sortKey)
	want := []any{2, 1}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("expected limit/offset args %v, got %v", want, args)
	}
}
