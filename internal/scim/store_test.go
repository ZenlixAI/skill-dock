package scim

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"skill_registry/internal/identity"
)

func TestSCIMProvisioningCreatesUserAndTenantMembership(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	identityStore := identity.NewSQLStore(db, "sqlite")
	if err := identityStore.Migrate(context.Background()); err != nil {
		t.Fatalf("identity migrate: %v", err)
	}
	store := NewStore(db, "sqlite", identityStore)

	user, err := store.CreateUser(context.Background(), User{
		UserName:    "ada@example.com",
		Email:       "ada@example.com",
		DisplayName: "Ada",
		Active:      true,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.ID == "" {
		t.Fatal("expected user id")
	}
	group, err := store.CreateGroup(context.Background(), Group{
		ExternalID:  "acme",
		DisplayName: "Acme",
		Members:     []Member{{Value: user.ID, Display: "Ada"}},
	})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if group.ID == "" {
		t.Fatal("expected group id")
	}
	organizations, err := identityStore.ListOrganizationsForUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("list organizations: %v", err)
	}
	if len(organizations) != 1 || organizations[0].Slug != "acme" {
		t.Fatalf("unexpected organizations: %+v", organizations)
	}
}
