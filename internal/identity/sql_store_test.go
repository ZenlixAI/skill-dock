package identity

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSQLStoreMigratesAndProjectsTenantMembership(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := NewSQLStore(db, "sqlite")
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	user, err := store.UpsertUser(context.Background(), User{
		Issuer:      "https://logto.example.com/oidc",
		Subject:     "user_123",
		Email:       "ada@example.com",
		DisplayName: "Ada",
	})
	if err != nil {
		t.Fatalf("upsert user: %v", err)
	}
	if user.ID == "" {
		t.Fatal("expected user id")
	}
	organization, err := store.UpsertOrganization(context.Background(), Organization{
		LogtoOrgID: "org_logto",
		Slug:       "acme",
		Name:       "Acme",
	})
	if err != nil {
		t.Fatalf("upsert organization: %v", err)
	}
	if err := store.UpsertMembership(context.Background(), Membership{
		OrganizationID: organization.ID,
		UserID:         user.ID,
		Roles:          []string{"admin"},
	}); err != nil {
		t.Fatalf("upsert membership: %v", err)
	}
	organizations, err := store.ListOrganizationsForUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("list organizations: %v", err)
	}
	if len(organizations) != 1 || organizations[0].Slug != "acme" || organizations[0].Role != "admin" {
		t.Fatalf("unexpected organizations: %+v", organizations)
	}
	member, err := store.IsMember(context.Background(), organization.ID, user.ID)
	if err != nil || !member {
		t.Fatalf("expected member, got %v %v", member, err)
	}
	if err := store.RemoveMembership(context.Background(), organization.ID, user.ID); err != nil {
		t.Fatalf("remove membership: %v", err)
	}
	member, err = store.IsMember(context.Background(), organization.ID, user.ID)
	if err != nil || member {
		t.Fatalf("expected removed member, got %v %v", member, err)
	}
}
