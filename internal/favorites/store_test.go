package favorites

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"skill_registry/internal/identity"
)

func TestFavoritesAreScopedByContext(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := identity.NewSQLStore(db, "sqlite").Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	store := NewStore(db, "sqlite")
	ctx := context.Background()

	for _, favorite := range []Favorite{
		{UserID: "usr_1", ContextType: "global", RefKey: "external:skills.sh:acme/repo:pdf", RefType: "external", ExternalProvider: "skills.sh", ExternalSource: "acme/repo", ExternalSkillID: "pdf", Snapshot: []byte(`{"name":"pdf"}`)},
		{UserID: "usr_1", ContextType: "tenant", OrganizationID: "org_1", RefKey: "catalog:@acme.skill/private", RefType: "catalog", PackageName: "@acme.skill/private", Snapshot: []byte(`{"name":"private"}`)},
		{UserID: "usr_1", ContextType: "personal", RefKey: "catalog:@draft.usr_1/notes", RefType: "catalog", PackageName: "@draft.usr_1/notes", Snapshot: []byte(`{"name":"notes"}`)},
		{UserID: "usr_2", ContextType: "tenant", OrganizationID: "org_1", RefKey: "catalog:@acme.skill/private", RefType: "catalog", PackageName: "@acme.skill/private", Snapshot: []byte(`{"name":"private"}`)},
	} {
		if _, err := store.Upsert(ctx, favorite); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}

	items, err := store.List(ctx, "usr_1", "org_1", true, 100, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected three favorites for usr_1 in org_1, got %d", len(items))
	}
	items, err = store.List(ctx, "usr_1", "org_2", false, 100, 0)
	if err != nil {
		t.Fatalf("list other tenant: %v", err)
	}
	if len(items) != 1 || items[0].RefKey != "external:skills.sh:acme/repo:pdf" {
		t.Fatalf("expected only global favorite in other tenant, got %+v", items)
	}
	if err := store.Delete(ctx, "usr_1", "external:skills.sh:acme/repo:pdf"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	status, err := store.Status(ctx, "usr_1", []string{"external:skills.sh:acme/repo:pdf"})
	if err != nil || status["external:skills.sh:acme/repo:pdf"] {
		t.Fatalf("expected deleted favorite to be absent: %+v %v", status, err)
	}
}
