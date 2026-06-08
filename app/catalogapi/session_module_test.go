package catalogapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"

	"skill_registry/app/auth"
)

func TestSessionModuleReturnsCurrentPrincipalView(t *testing.T) {
	module := newSessionModuleForTest(t)
	principal := auth.Principal{
		Role:            "publisher",
		Actor:           "svc-publisher",
		AuthMechanism:   "legacy_token",
		AllowedPackages: []string{"@platform.skill/code-review", "@platform.skill/agent-notes"},
	}

	response := module.current(principal)

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

func TestSessionEndpointRoutesThroughSessionModule(t *testing.T) {
	module := newSessionModuleStubForTest(t)
	server := newTestServerWithSessionModule(t, module)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	var response SessionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !reflect.DeepEqual(response.AllowedActions, []string{"delegated.action"}) {
		t.Fatalf("expected delegated session projection, got %v", response.AllowedActions)
	}
	if got := module.currentCallCount(); got != 1 {
		t.Fatalf("expected session module current to be called once, got %d", got)
	}
}

func TestSessionLoginRoutesThroughSessionModule(t *testing.T) {
	module := newSessionModuleStubForTest(t)
	server := newTestServerWithSessionModule(t, module)
	request := jsonRequest(http.MethodPost, "/api/v1/session/login", `{"username":"demo","password":"secret"}`)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if got := module.loginCallCount(); got != 1 {
		t.Fatalf("expected session module login to be called once, got %d", got)
	}
	if len(recorder.Result().Cookies()) == 0 {
		t.Fatalf("expected login cookie to be set")
	}
}

func TestSessionLogoutRoutesThroughSessionModule(t *testing.T) {
	module := newSessionModuleStubForTest(t)
	server := newTestServerWithSessionModule(t, module)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/session/logout", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if got := module.logoutCallCount(); got != 1 {
		t.Fatalf("expected session module logout to be called once, got %d", got)
	}
	if len(recorder.Result().Cookies()) == 0 {
		t.Fatalf("expected logout cookie to be set")
	}
}
