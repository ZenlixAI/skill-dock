package tags

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"skill_registry/internal/identity"
)

func TestTenantTagsAreVisibleOnlyInTheirTenant(t *testing.T) {
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
	if _, err := store.Create(ctx, Tag{Slug: "security", Label: "Security", Scope: "system", Visibility: "public"}); err != nil {
		t.Fatalf("create system tag: %v", err)
	}
	tenantTag, err := store.Create(ctx, Tag{Slug: "team-payments", Label: "Payments", Scope: "tenant", OrganizationID: "org_1", Visibility: "private"})
	if err != nil {
		t.Fatalf("create tenant tag: %v", err)
	}
	visible, err := store.ListVisible(ctx, "org_1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(visible) != 2 {
		t.Fatalf("expected two visible tags, got %+v", visible)
	}
	visible, err = store.ListVisible(ctx, "org_2")
	if err != nil {
		t.Fatalf("list other tenant: %v", err)
	}
	if len(visible) != 1 || visible[0].Slug != "security" {
		t.Fatalf("expected only the system tag for another tenant, got %+v", visible)
	}
	if err := store.ReplaceSkillTags(ctx, "org_1", "@acme.skill/pdf", []string{tenantTag.ID}, "usr_1"); err != nil {
		t.Fatalf("replace tags: %v", err)
	}
	assigned, err := store.ListForPackage(ctx, "@acme.skill/pdf")
	if err != nil || len(assigned) != 1 || assigned[0].ID != tenantTag.ID {
		t.Fatalf("unexpected assigned tags: %+v %v", assigned, err)
	}
}
