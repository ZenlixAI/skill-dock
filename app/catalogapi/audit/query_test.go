package audit

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"skill_registry/business/catalog"
	"skill_registry/internal/platform/auth"
	"skill_registry/internal/platform/observability"
	persist "skill_registry/internal/platform/persistence"
)

func TestAuditQueryModuleRejectsInvalidAuthDecision(t *testing.T) {
	module := &QueryModule{
		audit:      stubAuditStore{},
		authorizer: auth.NewAuthorizer(auth.Config{}),
	}

	_, err := module.ListEvents(context.Background(), auth.Principal{Role: "publisher", Actor: "publisher"}, url.Values{
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
	module := &QueryModule{
		audit:      stubAuditStore{},
		authorizer: auth.NewAuthorizer(auth.Config{}),
	}

	_, err := module.ListEvents(context.Background(), auth.Principal{
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
	module := &QueryModule{
		audit: stubAuditStore{
			listPage: persist.AuditPage{
				Items: []persist.AuditRecord{{
					Action:       "release.record",
					Actor:        "svc-publisher",
					Role:         "publisher",
					AuthDecision: observability.AuthDecisionAllowed,
					PackageName:  "@platform.skill/agent-notes",
					Version:      new("0.1.0"),
					Outcome:      "succeeded",
					StatusCode:   http.StatusCreated,
					CreatedAt:    time.Now().UTC(),
				}},
				PageInfo: catalog.PageInfo{},
			},
		},
		authorizer: auth.NewAuthorizer(auth.Config{}),
	}

	page, err := module.ListEvents(context.Background(), auth.Principal{Role: "publisher", Actor: "svc-publisher"}, url.Values{
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

func TestAuditQueryModuleReturnsServiceUnavailableWhenStoreMissing(t *testing.T) {
	module := &QueryModule{
		authorizer: auth.NewAuthorizer(auth.Config{}),
	}

	_, err := module.ListEvents(context.Background(), auth.Principal{Role: "publisher", Actor: "publisher"}, url.Values{})

	if err == nil {
		t.Fatal("expected missing store error")
	}
	if err.Status != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, err.Status)
	}
}

func TestAuditQueryModulePropagatesAuditStoreErrors(t *testing.T) {
	module := &QueryModule{
		audit: stubAuditStore{
			listErr: catalog.NewError(http.StatusServiceUnavailable, "audit_unavailable", "Audit backend unavailable.", nil),
		},
		authorizer: auth.NewAuthorizer(auth.Config{}),
	}

	_, err := module.ListEvents(context.Background(), auth.Principal{Role: "publisher", Actor: "publisher"}, url.Values{})

	if err == nil {
		t.Fatal("expected store error")
	}
	if err.Code != "audit_unavailable" {
		t.Fatalf("expected audit_unavailable code, got %+v", err)
	}
}

func TestAuditQueryModuleRejectsInvalidVersionFilter(t *testing.T) {
	module := &QueryModule{
		audit:      stubAuditStore{},
		authorizer: auth.NewAuthorizer(auth.Config{}),
	}

	_, err := module.ListEvents(context.Background(), auth.Principal{Role: "publisher", Actor: "publisher"}, url.Values{
		"version": []string{"latest"},
	})

	if err == nil {
		t.Fatal("expected invalid version error")
	}
	if err.Status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, err.Status)
	}
}

type stubAuditStore struct {
	appendErr error
	listPage  persist.AuditPage
	listErr   error
	pingErr   error
}

func (s stubAuditStore) Append(context.Context, persist.AuditEvent) error {
	return s.appendErr
}

func (s stubAuditStore) List(context.Context, persist.AuditQuery) (persist.AuditPage, error) {
	return s.listPage, s.listErr
}

func (s stubAuditStore) Ping(context.Context) error {
	return s.pingErr
}
