package httpserver

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"

	cataloginfra "skill_registry/internal/catalog/infra"
	"skill_registry/internal/identity"
	"skill_registry/internal/scim"
)

func TestSCIMRoutesProvisionUserAndGroup(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	identityStore := identity.NewSQLStore(db, "sqlite")
	if err := identityStore.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	scimStore := scim.NewStore(db, "sqlite", identityStore)
	server := NewServer(ServerConfig{
		Repository:    cataloginfra.NewMemoryRepository(cataloginfra.SampleData()),
		SCIMStore:     scimStore,
		SCIMToken:     "scim-secret",
		IdentityStore: identityStore,
	})
	srv := httptest.NewServer(server.Handler())
	defer srv.Close()

	userPayload := `{"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],"userName":"ada@example.com","name":{"formatted":"Ada"},"emails":[{"value":"ada@example.com","primary":true}],"active":true}`
	request, _ := http.NewRequest(http.MethodPost, srv.URL+"/scim/v2/Users", bytes.NewBufferString(userPayload))
	request.Header.Set("Authorization", "Bearer scim-secret")
	request.Header.Set("Content-Type", "application/scim+json")
	response, err := srv.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected user create 201, got %d", response.StatusCode)
	}
	var user struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(response.Body).Decode(&user)
	if user.ID == "" {
		t.Fatal("expected SCIM user id")
	}

	groupPayload := `{"schemas":["urn:ietf:params:scim:schemas:core:2.0:Group"],"externalId":"acme","displayName":"Acme","members":[{"value":"` + user.ID + `"}]}`
	request, _ = http.NewRequest(http.MethodPost, srv.URL+"/scim/v2/Groups", bytes.NewBufferString(groupPayload))
	request.Header.Set("Authorization", "Bearer scim-secret")
	request.Header.Set("Content-Type", "application/scim+json")
	response, err = srv.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("expected group create 201, got %d", response.StatusCode)
	}
	organizations, err := identityStore.ListOrganizationsForUser(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(organizations) != 1 || organizations[0].Slug != "acme" {
		t.Fatalf("unexpected organizations: %+v", organizations)
	}
}
