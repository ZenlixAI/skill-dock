package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	platformsession "skill_registry/internal/platform/session"
)

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
	var response platformsession.Response
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
