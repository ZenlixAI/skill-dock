package access

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"skill_registry/internal/platform/auth"
)

func TestDeciderPublicTenantGrantAndMismatch(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE skill_grants (id TEXT PRIMARY KEY, organization_id TEXT, package_name TEXT, subject_type TEXT, subject_id TEXT, permission TEXT, granted_by TEXT, expires_at TEXT, created_at TEXT, UNIQUE (organization_id, package_name, subject_type, subject_id))`,
		`CREATE TABLE skill_maintainers (package_name TEXT, organization_id TEXT, user_id TEXT, granted_by TEXT, created_at TEXT, PRIMARY KEY (package_name, user_id))`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	store := NewSQLStore(db, "sqlite")
	decider := NewDecider(store)
	ctx := context.Background()

	public := SkillAccess{PackageName: "@acme.skill/public", Visibility: "public"}
	if decision := decider.Decide(ctx, auth.Principal{}, ActionRead, public); !decision.Allowed {
		t.Fatalf("anonymous public read should be allowed: %+v", decision)
	}
	private := SkillAccess{PackageName: "@acme.skill/private", OrganizationID: "org_1", OwnerType: "organization", Visibility: "private"}
	if decision := decider.Decide(ctx, auth.Principal{}, ActionRead, private); decision.Allowed {
		t.Fatal("anonymous private read should be denied")
	}
	member := auth.Principal{Kind: auth.PrincipalKindUser, UserID: "usr_1", TenantID: "org_1"}
	orgVisible := SkillAccess{PackageName: "@acme.skill/org", OrganizationID: "org_1", OwnerType: "organization", Visibility: "organization"}
	if decision := decider.Decide(ctx, member, ActionRead, orgVisible); !decision.Allowed {
		t.Fatalf("tenant member should read organization skill: %+v", decision)
	}
	otherTenant := auth.Principal{Kind: auth.PrincipalKindUser, UserID: "usr_2", TenantID: "org_2"}
	if decision := decider.Decide(ctx, otherTenant, ActionRead, orgVisible); decision.Allowed {
		t.Fatal("other tenant should be denied")
	}

	if err := store.PutGrant(ctx, Grant{
		OrganizationID: "org_1",
		PackageName:    private.PackageName,
		SubjectType:    "user",
		SubjectID:      "usr_2",
		Permission:     "view",
	}); err != nil {
		t.Fatal(err)
	}
	if decision := decider.Decide(ctx, otherTenant, ActionRead, private); decision.Allowed {
		t.Fatal("cross-tenant grant principal should still be denied by tenant mismatch")
	}
	granted := auth.Principal{Kind: auth.PrincipalKindUser, UserID: "usr_3", TenantID: "org_1"}
	if err := store.PutGrant(ctx, Grant{
		OrganizationID: "org_1",
		PackageName:    private.PackageName,
		SubjectType:    "user",
		SubjectID:      "usr_3",
		Permission:     "install",
	}); err != nil {
		t.Fatal(err)
	}
	if decision := decider.Decide(ctx, granted, ActionRead, private); !decision.Allowed {
		t.Fatalf("granted user should read private skill: %+v", decision)
	}
	if decision := decider.Decide(ctx, granted, ActionManage, private); decision.Allowed {
		t.Fatal("install grant should not allow manage")
	}
	expired := time.Now().Add(-time.Minute)
	if err := store.PutGrant(ctx, Grant{
		OrganizationID: "org_1",
		PackageName:    "@acme.skill/expired",
		SubjectType:    "user",
		SubjectID:      "usr_4",
		Permission:     "view",
		ExpiresAt:      &expired,
	}); err != nil {
		t.Fatal(err)
	}
	expiredPrincipal := auth.Principal{Kind: auth.PrincipalKindUser, UserID: "usr_4", TenantID: "org_1"}
	if decision := decider.Decide(ctx, expiredPrincipal, ActionRead, SkillAccess{PackageName: "@acme.skill/expired", OrganizationID: "org_1", OwnerType: "organization", Visibility: "private"}); decision.Allowed {
		t.Fatal("expired grant should not allow access")
	}
}
