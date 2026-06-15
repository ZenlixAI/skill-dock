package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthorizerAuthenticateUsesConfiguredBearerToken(t *testing.T) {
	authorizer := NewAuthorizer(Config{
		Tokens: map[string]TokenConfig{
			"reader-secret": {
				Role:  "reader",
				Actor: "svc-reader",
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	request.Header.Set("Authorization", "Bearer reader-secret")
	request.Header.Set("X-Role", "publisher")

	current, ok := authorizer.Authenticate(request)
	if !ok {
		t.Fatal("expected authentication to succeed")
	}
	if current.Role != "reader" {
		t.Fatalf("expected reader role, got %s", current.Role)
	}
	if current.Actor != "svc-reader" {
		t.Fatalf("expected configured actor, got %s", current.Actor)
	}
}

func TestAuthorizerAuthorizePackageRejectsUnauthorizedPackage(t *testing.T) {
	authorizer := NewAuthorizer(Config{})
	current := Principal{
		Role:            "publisher",
		Actor:           "svc-publisher",
		AuthMechanism:   "configured_token",
		AllowedPackages: []string{"@platform.skill/agent-notes"},
	}

	err := authorizer.AuthorizePackage(current, "@platform.skill/pdf-skill")
	if err == nil {
		t.Fatal("expected package authorization error")
	}
	if err.Code != "package_forbidden" {
		t.Fatalf("expected package_forbidden code, got %s", err.Code)
	}
}

func TestAuthorizerAuthorizeRoleRejectsDisallowedRole(t *testing.T) {
	authorizer := NewAuthorizer(Config{})
	current := Principal{Role: "reader", Actor: "reader"}

	err := authorizer.AuthorizeRole(current, "publisher")
	if err == nil {
		t.Fatal("expected role authorization error")
	}
	if err.Code != "forbidden" {
		t.Fatalf("expected forbidden code, got %s", err.Code)
	}
}

func TestAuthorizerAuthorizeActionUsesCapabilityMatrix(t *testing.T) {
	authorizer := NewAuthorizer(Config{})

	if err := authorizer.AuthorizeAction(Principal{Role: "reader", Actor: "reader"}, ActionCreateReleaseRecord); err == nil {
		t.Fatal("expected reader to be forbidden from creating release records")
	}
	if err := authorizer.AuthorizeAction(Principal{Role: "authoring", Actor: "authoring"}, ActionPackagingPrepare); err != nil {
		t.Fatalf("expected authoring to prepare packaging, got %v", err)
	}
	if err := authorizer.AuthorizeAction(Principal{Role: "authoring", Actor: "authoring"}, ActionApplySkillGovernance); err == nil {
		t.Fatal("expected authoring to be forbidden from skill governance")
	}
	if err := authorizer.AuthorizeAction(Principal{Role: "publisher", Actor: "publisher"}, ActionApplyVersionGovernance); err != nil {
		t.Fatalf("expected publisher to apply version governance, got %v", err)
	}
}
