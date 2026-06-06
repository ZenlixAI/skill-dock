package catalogapi

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"skill_registry/app/auth"
	"skill_registry/app/observability"
	"skill_registry/business/catalog"
	"skill_registry/foundation/persist"
)

func TestAuditQueryModuleRejectsInvalidAuthDecision(t *testing.T) {
	module := &auditQueryModule{
		audit:      stubAuditStore{},
		authorizer: auth.NewAuthorizer(auth.Config{}),
	}

	_, err := module.listEvents(context.Background(), auth.Principal{Role: "publisher", Actor: "publisher"}, url.Values{
		"auth_decision": []string{"maybe"},
	})

	if err == nil {
		t.Fatal("expected invalid auth_decision error")
	}
	if err.Status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, err.Status)
	}
}

func TestAuditQueryModuleRequiresPackageFilterForScopedPrincipal(t *testing.T) {
	module := &auditQueryModule{
		audit:      stubAuditStore{},
		authorizer: auth.NewAuthorizer(auth.Config{}),
	}

	_, err := module.listEvents(context.Background(), auth.Principal{
		Role:            "reader",
		Actor:           "svc-reader",
		AllowedPackages: []string{"@platform.skill/private-skill"},
	}, url.Values{})

	if err == nil {
		t.Fatal("expected package filter requirement error")
	}
	if err.Status != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, err.Status)
	}
}

func TestAuditQueryModuleListsEventsWithCombinedFilters(t *testing.T) {
	module := &auditQueryModule{
		audit: stubAuditStore{
			listPage: persist.AuditPage{
				Items: []persist.AuditRecord{{
					Action:       "release.record",
					Actor:        "svc-publisher",
					Role:         "publisher",
					AuthDecision: observability.AuthDecisionAllowed,
					PackageName:  "@platform.skill/agent-notes",
					Version:      stringPtr("0.1.0"),
					Outcome:      "succeeded",
					StatusCode:   http.StatusCreated,
					CreatedAt:    time.Now().UTC(),
				}},
				PageInfo: catalog.PageInfo{},
			},
		},
		authorizer: auth.NewAuthorizer(auth.Config{}),
	}

	page, err := module.listEvents(context.Background(), auth.Principal{Role: "publisher", Actor: "svc-publisher"}, url.Values{
		"action":        []string{"release.record"},
		"auth_decision": []string{"allowed"},
		"package_name":  []string{"@platform.skill/agent-notes"},
		"version":       []string{"0.1.0"},
		"outcome":       []string{"succeeded"},
	})

	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected one audit event, got %d", len(page.Items))
	}
	if page.Items[0].Action != "release.record" || page.Items[0].Actor != "svc-publisher" {
		t.Fatalf("unexpected audit event: %+v", page.Items[0])
	}
}
