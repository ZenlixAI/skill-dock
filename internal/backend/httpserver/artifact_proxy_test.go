package httpserver

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"

	"skill_registry/internal/access"
	"skill_registry/internal/identity"
	"skill_registry/internal/platform/auth"
)

func TestPackageFromArtifactPath(t *testing.T) {
	cases := map[string]string{
		"/repository/skills-npm/@acme.skill%2Fpdf/-/pdf-1.4.0.tgz": "@acme.skill/pdf",
		"/repository/skills-npm/@acme.skill/pdf":                   "@acme.skill/pdf",
		"/repository/skills-npm/skills/-/skills-1.6.9.tgz":         "skills",
	}
	for input, expected := range cases {
		actual, ok := packageFromArtifactPath(input)
		if !ok || actual != expected {
			t.Fatalf("packageFromArtifactPath(%q) = %q, %v; want %q", input, actual, ok, expected)
		}
	}
}

func TestAuthenticatedArtifactProxyEnforcesTenantAndGrant(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := identity.NewSQLStore(db, "sqlite").Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	store := access.NewSQLStore(db, "sqlite")
	if err := store.PutSkillAccess(t.Context(), access.SkillAccess{
		PackageName:    "@acme.skill/private",
		OrganizationID: "org_1",
		OwnerType:      "organization",
		OwnerID:        "org_1",
		Visibility:     "private",
	}); err != nil {
		t.Fatal(err)
	}
	recordCheck, err := store.GetSkillAccess(t.Context(), "@acme.skill/private")
	if err != nil || recordCheck.Visibility != "private" {
		t.Fatalf("unexpected access record: %+v %v", recordCheck, err)
	}
	if err := store.PutGrant(t.Context(), access.Grant{
		OrganizationID: "org_1",
		PackageName:    "@acme.skill/private",
		SubjectType:    "user",
		SubjectID:      "usr_1",
		Permission:     "install",
	}); err != nil {
		t.Fatal(err)
	}
	server := &Server{accessStore: store, access: access.NewDecider(store)}
	next := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { writer.WriteHeader(http.StatusOK) })
	privateNext := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("X-Private", "1")
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte("private"))
	})
	handler := server.authenticatedArtifactProxy(next, privateNext)

	requestURL := "/repository/skills-npm/%40acme.skill%2Fprivate/-/private-1.0.0.tgz"
	request := httptest.NewRequest(http.MethodGet, requestURL, nil)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	handler(context)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected anonymous 401, got %d", recorder.Code)
	}

	request = httptest.NewRequest(http.MethodGet, requestURL, nil)
	request = request.WithContext(auth.ContextWithPrincipal(request.Context(), auth.Principal{
		Kind: auth.PrincipalKindUser, UserID: "usr_1", TenantID: "org_1",
	}))
	recorder = httptest.NewRecorder()
	context, _ = gin.CreateTestContext(recorder)
	context.Request = request
	handler(context)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected granted private proxy 201, got %d header=%q", recorder.Code, recorder.Header().Get("X-Private"))
	}

	request = httptest.NewRequest(http.MethodGet, requestURL, nil)
	request = request.WithContext(auth.ContextWithPrincipal(request.Context(), auth.Principal{
		Kind: auth.PrincipalKindUser, UserID: "usr_1", TenantID: "org_2",
	}))
	recorder = httptest.NewRecorder()
	context, _ = gin.CreateTestContext(recorder)
	context.Request = request
	handler(context)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected cross-tenant 403, got %d", recorder.Code)
	}
}
