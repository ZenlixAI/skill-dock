package session

import (
	"reflect"
	"sort"
	"testing"

	"skill_registry/internal/platform/auth"
)

func TestModuleReturnsCurrentPrincipalView(t *testing.T) {
	module := NewModule(auth.NewAuthorizer(auth.Config{
		PublisherToken: "publisher-secret",
	}), nil)
	principal := auth.Principal{
		Role:            "publisher",
		Actor:           "svc-publisher",
		AuthMechanism:   "legacy_token",
		AllowedPackages: []string{"@platform.skill/code-review", "@platform.skill/agent-notes"},
	}

	response := module.Current(principal)

	if response.Role != "publisher" {
		t.Fatalf("expected role publisher, got %q", response.Role)
	}
	if response.Actor != "svc-publisher" {
		t.Fatalf("expected actor svc-publisher, got %q", response.Actor)
	}
	if response.AuthMechanism != "legacy_token" {
		t.Fatalf("expected auth mechanism legacy_token, got %q", response.AuthMechanism)
	}
	expectedActions := []string{
		"audit.read",
		"catalog.governance.skill.apply",
		"catalog.governance.version.apply",
		"catalog.read",
		"catalog.release_record.create",
		"packaging.pack",
		"packaging.prepare",
		"packaging.publish",
		"packaging.validate",
		"session.read",
	}
	actualActions := append([]string(nil), response.AllowedActions...)
	sort.Strings(expectedActions)
	sort.Strings(actualActions)
	if !reflect.DeepEqual(actualActions, expectedActions) {
		t.Fatalf("expected live allowed actions semantics, got %v", response.AllowedActions)
	}
	if !reflect.DeepEqual(response.AllowedPackages, []string{"@platform.skill/code-review", "@platform.skill/agent-notes"}) {
		t.Fatalf("expected allowed packages projection, got %v", response.AllowedPackages)
	}
}
