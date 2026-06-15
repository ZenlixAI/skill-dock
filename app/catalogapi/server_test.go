package catalogapi

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/binding"
	"skill_registry/app/catalogapi/common"
	"skill_registry/app/observability"
	"skill_registry/business/catalog"
	"skill_registry/business/packaging"
	artifactadapter "skill_registry/foundation/artifact"
	"skill_registry/foundation/persist"
	platformhttp "skill_registry/internal/platform/http"
)

func TestSkillsRequiresAuthentication(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
	if recorder.Header().Get("X-Request-Id") == "" {
		t.Fatal("expected X-Request-Id response header")
	}
	if recorder.Header().Get("X-Trace-Id") == "" {
		t.Fatal("expected X-Trace-Id response header")
	}

	assertErrorCode(t, recorder, "unauthorized")
}

func TestServerRoutesSkillsThroughCatalogReadModule(t *testing.T) {
	module := newCatalogReadModuleStubForTest(t)
	server := newTestServer()
	server.catalogReadModule = module
	server.readAPIModule = &readAPIModule{
		catalogReadModule: module,
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if got := module.listSkillsCallCount(); got != 1 {
		t.Fatalf("expected catalog read module to handle one list request, got %d", got)
	}
}

func TestServerRoutesHeadSkillsThroughCatalogReadModule(t *testing.T) {
	module := newCatalogReadModuleStubForTest(t)
	server := newTestServer()
	server.catalogReadModule = module
	server.readAPIModule = &readAPIModule{
		catalogReadModule: module,
	}
	request := httptest.NewRequest(http.MethodHead, "/api/v1/skills", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if got := module.listSkillsCallCount(); got != 1 {
		t.Fatalf("expected catalog read module to handle one HEAD list request, got %d", got)
	}
}

func TestServerRoutesEncodedPackageDetailThroughHandler(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository:  catalog.NewMemoryRepository(catalog.SampleData()),
		ReaderToken: "reader-secret",
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fpdf-skill", nil)
	request.Header.Set("Authorization", "Bearer reader-secret")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response struct {
		PackageName string `json:"packageName"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.PackageName != "@platform.skill/pdf-skill" {
		t.Fatalf("expected decoded package name, got %q", response.PackageName)
	}
}

func TestPlatformEnginePreservesRawPathSettings(t *testing.T) {
	var seamEngine any
	platformhttp.TestNewEngineHook = func(engine *gin.Engine) {
		seamEngine = engine
	}
	defer func() {
		platformhttp.TestNewEngineHook = nil
	}()

	server := newTestServer()
	handler := server.Handler()

	engine, ok := handler.(*gin.Engine)
	if !ok {
		t.Fatalf("expected *gin.Engine handler, got %T", handler)
	}
	if seamEngine != engine {
		t.Fatal("expected server handler to be constructed through platformhttp.NewEngine")
	}
	if !engine.UseRawPath {
		t.Fatal("expected UseRawPath enabled")
	}
	if engine.UnescapePathValues {
		t.Fatal("expected UnescapePathValues disabled")
	}
	if !engine.HandleMethodNotAllowed {
		t.Fatal("expected HandleMethodNotAllowed enabled")
	}
	if engine.RedirectTrailingSlash {
		t.Fatal("expected RedirectTrailingSlash disabled")
	}
	if engine.RedirectFixedPath {
		t.Fatal("expected RedirectFixedPath disabled")
	}
}

func TestReadAPIModuleListsSkillsFromParsedQuery(t *testing.T) {
	workflow := &capturingCatalogReadWorkflow{
		response: catalog.SkillsPage{
			Items: []catalog.SkillSummary{
				{PackageName: "@platform.skill/pdf-skill"},
			},
		},
	}
	module := &readAPIModule{
		catalogReadModule: workflow,
	}
	principal := auth.Principal{
		Role:  "reader",
		Actor: "reader",
	}

	response, err := module.listSkills(context.Background(), principal, binding.SkillsQuery{
		Query:          "pdf",
		PackageName:    "@platform.skill/pdf-skill",
		SourceType:     string(catalog.SourceTypeImported),
		IncludeRetired: "true",
		Sort:           "package_name_asc",
		PageSize:       "7",
		Cursor:         "cursor_12",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got := workflow.listSkillsCallCount(); got != 1 {
		t.Fatalf("expected catalog read workflow to be called once, got %d", got)
	}
	if !reflect.DeepEqual(workflow.principal, principal) {
		t.Fatalf("unexpected principal (-want +got):\nwant: %#v\ngot:  %#v", principal, workflow.principal)
	}

	wantQuery := catalog.SkillQuery{
		Query:          "pdf",
		PackageName:    "@platform.skill/pdf-skill",
		SourceType:     string(catalog.SourceTypeImported),
		IncludeRetired: true,
		Sort:           "package_name_asc",
		PageSize:       7,
		Cursor:         12,
	}
	if !reflect.DeepEqual(workflow.query, wantQuery) {
		t.Fatalf("unexpected parsed query (-want +got):\nwant: %#v\ngot:  %#v", wantQuery, workflow.query)
	}
	if !reflect.DeepEqual(response, workflow.response) {
		t.Fatalf("expected response passthrough, got %#v", response)
	}
}

func TestBearerTokenAuthUsesConfiguredRoleMapping(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository:     catalog.NewMemoryRepository(catalog.SampleData()),
		ReaderToken:    "reader-secret",
		PublisherToken: "publisher-secret",
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	request.Header.Set("Authorization", "Bearer reader-secret")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestBearerTokenAuthRejectsUnknownTokenWhenMappingConfigured(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository:     catalog.NewMemoryRepository(catalog.SampleData()),
		ReaderToken:    "reader-secret",
		PublisherToken: "publisher-secret",
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	request.Header.Set("Authorization", "Bearer reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	assertErrorCode(t, recorder, "unauthorized")
}

func TestBearerTokenAuthRejectsXRoleOverrideWhenMappingConfigured(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository:     catalog.NewMemoryRepository(catalog.SampleData()),
		ReaderToken:    "reader-secret",
		PublisherToken: "publisher-secret",
	})

	request := httptest.NewRequest(http.MethodPut, "/api/v1/skills/%40platform.skill%2Fagent-notes/versions/0.1.0/release-record", nil)
	request.Header.Set("Authorization", "Bearer reader-secret")
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusForbidden, recorder.Code, recorder.Body.String())
	}

	assertErrorCode(t, recorder, "forbidden")
}

func TestSessionEndpointReturnsCurrentPrincipal(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository:     catalog.NewMemoryRepository(catalog.SampleData()),
		ReaderToken:    "reader-secret",
		PublisherToken: "publisher-secret",
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	request.Header.Set("Authorization", "Bearer publisher-secret")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var raw map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw response: %v", err)
	}
	allowedPackagesRaw, ok := raw["allowedPackages"]
	if !ok {
		t.Fatalf("expected allowedPackages field in response body: %s", recorder.Body.String())
	}
	allowedPackagesArray, ok := allowedPackagesRaw.([]any)
	if !ok {
		t.Fatalf("expected allowedPackages to be a JSON array, got %T", allowedPackagesRaw)
	}

	var response struct {
		Role            string   `json:"role"`
		Actor           string   `json:"actor"`
		AuthMechanism   string   `json:"authMechanism"`
		AllowedActions  []string `json:"allowedActions"`
		AllowedPackages []string `json:"allowedPackages"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Role != "publisher" {
		t.Fatalf("expected role publisher, got %q", response.Role)
	}
	if response.Actor != "publisher" {
		t.Fatalf("expected actor publisher, got %q", response.Actor)
	}
	if response.AuthMechanism != "legacy_token" {
		t.Fatalf("expected auth mechanism legacy_token, got %q", response.AuthMechanism)
	}
	if len(response.AllowedActions) == 0 {
		t.Fatal("expected allowed actions to be returned")
	}
	if len(response.AllowedPackages) != 0 {
		t.Fatalf("expected no package restrictions, got %v", response.AllowedPackages)
	}
	if len(allowedPackagesArray) != 0 {
		t.Fatalf("expected allowedPackages JSON array to be empty, got %v", allowedPackagesArray)
	}
}

func TestSessionEndpointRequiresAuthentication(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if strings.TrimSpace(recorder.Body.String()) != "null" {
		t.Fatalf("expected null session body, got %s", recorder.Body.String())
	}
}

func TestSessionEndpointAllowsAuthoringPrincipal(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		AuthTokens: map[string]auth.TokenConfig{
			"authoring-secret": {
				Role:  "authoring",
				Actor: "svc-authoring",
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	request.Header.Set("Authorization", "Bearer authoring-secret")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response struct {
		Role           string   `json:"role"`
		AllowedActions []string `json:"allowedActions"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Role != "authoring" {
		t.Fatalf("expected role authoring, got %q", response.Role)
	}
	if !reflect.DeepEqual(response.AllowedActions, []string{"packaging.pack", "packaging.prepare", "packaging.validate", "session.read"}) {
		t.Fatalf("unexpected allowed actions: %v", response.AllowedActions)
	}
}

func TestSessionEndpointHandlesLocalOperatorPreflight(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
	})

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/session", nil)
	request.Header.Set("Origin", "http://127.0.0.1:4174")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNoContent, recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:4174" {
		t.Fatalf("expected allow origin header, got %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Authorization") {
		t.Fatalf("expected Authorization in allow headers, got %q", got)
	}
}

func TestSkillsEndpointHandlesLocalOperatorPreflight(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
	})

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/skills", nil)
	request.Header.Set("Origin", "http://127.0.0.1:4174")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNoContent, recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:4174" {
		t.Fatalf("expected allow origin header, got %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Authorization") {
		t.Fatalf("expected Authorization in allow headers, got %q", got)
	}
}

func TestSessionEndpointAddsCORSHeadersForLocalOperatorOrigin(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository:     catalog.NewMemoryRepository(catalog.SampleData()),
		PublisherToken: "publisher-secret",
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	request.Header.Set("Origin", "http://localhost:4174")
	request.Header.Set("Authorization", "Bearer publisher-secret")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:4174" {
		t.Fatalf("expected allow origin header, got %q", got)
	}
}

func TestSessionEndpointReturnsAllowedPackagesForScopedPrincipal(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		AuthTokens: map[string]auth.TokenConfig{
			"publisher-secret": {
				Role:            "publisher",
				Actor:           "svc-publisher",
				AllowedPackages: []string{"@platform.skill/code-review", "@platform.skill/agent-notes"},
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	request.Header.Set("Authorization", "Bearer publisher-secret")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response struct {
		AllowedPackages []string `json:"allowedPackages"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if !reflect.DeepEqual(response.AllowedPackages, []string{"@platform.skill/code-review", "@platform.skill/agent-notes"}) {
		t.Fatalf("unexpected allowed packages: %v", response.AllowedPackages)
	}
}

func TestSessionLoginSetsCookieAndReturnsOperatorSession(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository:    catalog.NewMemoryRepository(catalog.SampleData()),
		SessionSecret: "session-secret",
		SessionUsers: map[string]auth.SessionUserConfig{
			"operator": {
				Password: "op-secret",
				Role:     "operator",
			},
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/session/login", strings.NewReader(`{"username":"operator","password":"op-secret"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie to be set")
	}

	var response struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Role != "operator" {
		t.Fatalf("expected role operator, got %q", response.Role)
	}
}

func TestSessionLoginRejectsInvalidJSON(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository:    catalog.NewMemoryRepository(catalog.SampleData()),
		SessionSecret: "session-secret",
		SessionUsers: map[string]auth.SessionUserConfig{
			"operator": {
				Password: "op-secret",
				Role:     "operator",
			},
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/session/login", strings.NewReader(`{"username":"operator"`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
}

func TestSessionEndpointAcceptsCookieAuthentication(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository:    catalog.NewMemoryRepository(catalog.SampleData()),
		SessionSecret: "session-secret",
		SessionUsers: map[string]auth.SessionUserConfig{
			"reader": {
				Password: "reader-secret",
				Role:     "reader",
			},
		},
	})

	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/session/login", strings.NewReader(`{"username":"reader","password":"reader-secret"}`))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(loginRecorder, loginRequest)
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login status %d, got %d body=%s", http.StatusOK, loginRecorder.Code, loginRecorder.Body.String())
	}
	cookies := loginRecorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie after login")
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/session", nil)
	request.AddCookie(cookies[0])
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response struct {
		Role           string   `json:"role"`
		AuthMechanism  string   `json:"authMechanism"`
		AllowedActions []string `json:"allowedActions"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Role != "reader" {
		t.Fatalf("expected role reader, got %q", response.Role)
	}
	if response.AuthMechanism != "session_cookie" {
		t.Fatalf("expected session_cookie auth, got %q", response.AuthMechanism)
	}
	if !reflect.DeepEqual(response.AllowedActions, []string{"audit.read", "catalog.read", "session.read"}) {
		t.Fatalf("unexpected allowed actions: %v", response.AllowedActions)
	}
}

func TestSessionLogoutClearsCookie(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository:    catalog.NewMemoryRepository(catalog.SampleData()),
		SessionSecret: "session-secret",
		SessionUsers: map[string]auth.SessionUserConfig{
			"reader": {
				Password: "reader-secret",
				Role:     "reader",
			},
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/session/logout", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected logout cookie to be written")
	}
	if cookies[0].MaxAge != -1 {
		t.Fatalf("expected logout cookie max age -1, got %d", cookies[0].MaxAge)
	}
}

func TestPackageLevelAuthzRejectsUnauthorizedReleaseRecord(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		AuthTokens: map[string]auth.TokenConfig{
			"publisher-secret": {
				Role:            "publisher",
				Actor:           "svc-publisher",
				AllowedPackages: []string{"@platform.skill/agent-notes"},
			},
		},
	})

	request := httptest.NewRequest(http.MethodPut, "/api/v1/skills/%40platform.skill%2Fpdf-skill/versions/1.2.0/release-record", nil)
	request.Header.Set("Authorization", "Bearer publisher-secret")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusForbidden, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "package_forbidden")
	assertErrorDetail(t, recorder, "packageName", "@platform.skill/pdf-skill")
}

func TestPackageLevelAuthzAllowsAuthorizedPackageReadAndWrite(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		AuthTokens: map[string]auth.TokenConfig{
			"publisher-secret": {
				Role:            "publisher",
				Actor:           "svc-publisher",
				AllowedPackages: []string{"@platform.skill/agent-notes", "@platform.skill/code-review"},
			},
		},
	})

	releaseRequest := httptest.NewRequest(http.MethodPut, "/api/v1/skills/%40platform.skill%2Fagent-notes/versions/0.1.0/release-record", nil)
	releaseRequest.Header.Set("Authorization", "Bearer publisher-secret")
	releaseRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(releaseRecorder, releaseRequest)
	if releaseRecorder.Code != http.StatusCreated {
		t.Fatalf("expected release status %d, got %d body=%s", http.StatusCreated, releaseRecorder.Code, releaseRecorder.Body.String())
	}

	readRequest := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fcode-review", nil)
	readRequest.Header.Set("Authorization", "Bearer publisher-secret")
	readRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(readRecorder, readRequest)
	if readRecorder.Code != http.StatusOK {
		t.Fatalf("expected read status %d, got %d body=%s", http.StatusOK, readRecorder.Code, readRecorder.Body.String())
	}
}

func TestPackageScopedSkillsListReturnsAuthorizedPageWindow(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		AuthTokens: map[string]auth.TokenConfig{
			"reader-secret": {
				Role:            "reader",
				Actor:           "scoped-reader",
				AllowedPackages: []string{"@platform.skill/pdf-skill"},
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills?sort=package_name_asc&pageSize=1", nil)
	request.Header.Set("Authorization", "Bearer reader-secret")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response struct {
		Items    []catalog.SkillSummary `json:"items"`
		PageInfo catalog.PageInfo       `json:"pageInfo"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("expected one authorized item, got %d (%#v)", len(response.Items), response.Items)
	}
	if response.Items[0].PackageName != "@platform.skill/pdf-skill" {
		t.Fatalf("expected authorized package @platform.skill/pdf-skill, got %q", response.Items[0].PackageName)
	}
	if response.PageInfo.HasNextPage || response.PageInfo.NextCursor != nil {
		t.Fatalf("expected authorized page window to be terminal, got %+v", response.PageInfo)
	}
}

func TestGovernanceUsesConfiguredActorInsteadOfSpoofableHeader(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		AuthTokens: map[string]auth.TokenConfig{
			"publisher-secret": {
				Role:  "publisher",
				Actor: "svc-publisher",
			},
		},
	})

	request := jsonRequest(http.MethodPost, "/api/v1/skills/%40platform.skill%2Fcode-review/governance-changes", `{"action":"retire","reason":"merged into new pack"}`)
	request.Header.Set("Authorization", "Bearer publisher-secret")
	request.Header.Set("X-Actor", "user_789")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var response catalog.GovernanceChangeResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Record.ChangedBy != "svc-publisher" {
		t.Fatalf("expected actor svc-publisher, got %s", response.Record.ChangedBy)
	}
}

func TestMockRegistryRouteIsNotRegistered(t *testing.T) {
	server := newTestServer()

	for _, route := range []string{
		"/mock-npm",
		"/mock-npm/%40platform.skill%2Fpdf-skill",
	} {
		t.Run(route, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, route, nil)
			recorder := httptest.NewRecorder()

			server.Handler().ServeHTTP(recorder, request)

			if recorder.Code != http.StatusNotFound {
				t.Fatalf("expected status %d, got %d body=%s", http.StatusNotFound, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestServerCanonicalizesRepeatedSlashesBeforeRouting(t *testing.T) {
	server := newTestServer()

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills//foo", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusTemporaryRedirect, recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Location"); got != "/api/v1/skills/foo" {
		t.Fatalf("expected redirect location /api/v1/skills/foo, got %q", got)
	}
}

func TestServerCanonicalRedirectIncludesMetadataAndLocalCORS(t *testing.T) {
	var logBuffer bytes.Buffer
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:  packaging.NewService(""),
		Logger:     log.New(&logBuffer, "", 0),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills//foo", nil)
	request.Header.Set("Origin", "http://127.0.0.1:4174")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusTemporaryRedirect, recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Location"); got != "/api/v1/skills/foo" {
		t.Fatalf("expected redirect location /api/v1/skills/foo, got %q", got)
	}

	requestID := recorder.Header().Get("X-Request-Id")
	if requestID == "" {
		t.Fatal("expected X-Request-Id response header")
	}

	traceID := recorder.Header().Get("X-Trace-Id")
	if traceID == "" {
		t.Fatal("expected X-Trace-Id response header")
	}

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:4174" {
		t.Fatalf("expected allow origin header, got %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Authorization") {
		t.Fatalf("expected Authorization in allow headers, got %q", got)
	}

	payload := findLoggedEvent(t, logBuffer.String(), "http_request")
	if payload["status"] != float64(http.StatusTemporaryRedirect) {
		t.Fatalf("expected redirect status in request log, got %+v", payload)
	}
	if payload["path"] != "/api/v1/skills//foo" {
		t.Fatalf("expected original redirect path in request log, got %+v", payload)
	}
	if payload["requestId"] != requestID {
		t.Fatalf("expected logged requestId %q, got %+v", requestID, payload)
	}
	if payload["traceId"] != traceID {
		t.Fatalf("expected logged traceId %q, got %+v", traceID, payload)
	}
}

func TestSkillPackagingPrepareValidateAndPack(t *testing.T) {
	server := newTestServer()
	fixtureDir := createPackagingFixture(t)

	prepareRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":`+uploadArchiveSourceJSON(t, fixtureDir)+`,
		"target":{"packageName":"@platform.skill/fixture-skill","version":"0.2.0"},
		"metadata":{"displayName":"Fixture Skill","description":"Fixture packaging flow","sourceType":"imported"},
		"options":{"publishRegistry":"https://registry.example.com"}
	}`)
	prepareRequest.Header.Set("X-Role", "publisher")
	prepareRecorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(prepareRecorder, prepareRequest)

	if prepareRecorder.Code != http.StatusOK {
		t.Fatalf("expected prepare status %d, got %d body=%s", http.StatusOK, prepareRecorder.Code, prepareRecorder.Body.String())
	}

	var preparePayload struct {
		WorkspacePath string   `json:"workspacePath"`
		PackageName   string   `json:"packageName"`
		Version       string   `json:"version"`
		Warnings      []string `json:"warnings"`
		Errors        []string `json:"errors"`
	}
	if err := json.Unmarshal(prepareRecorder.Body.Bytes(), &preparePayload); err != nil {
		t.Fatalf("decode prepare response: %v", err)
	}
	if preparePayload.WorkspacePath == "" {
		t.Fatal("expected workspacePath to be returned")
	}

	validateRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/validate", `{
		"workspacePath":"`+preparePayload.WorkspacePath+`",
		"target":{"packageName":"@platform.skill/fixture-skill","version":"0.2.0"}
	}`)
	validateRequest.Header.Set("X-Role", "publisher")
	validateRecorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(validateRecorder, validateRequest)

	if validateRecorder.Code != http.StatusOK {
		t.Fatalf("expected validate status %d, got %d body=%s", http.StatusOK, validateRecorder.Code, validateRecorder.Body.String())
	}

	var validatePayload struct {
		OK         bool     `json:"ok"`
		Components []string `json:"components"`
		Errors     []string `json:"errors"`
	}
	if err := json.Unmarshal(validateRecorder.Body.Bytes(), &validatePayload); err != nil {
		t.Fatalf("decode validate response: %v", err)
	}
	if !validatePayload.OK {
		t.Fatalf("expected validate ok response, got %+v", validatePayload)
	}

	packRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/pack", `{
		"workspacePath":"`+preparePayload.WorkspacePath+`",
		"target":{"packageName":"@platform.skill/fixture-skill","version":"0.2.0"},
		"includeBundle":true
	}`)
	packRequest.Header.Set("X-Role", "publisher")
	packRecorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(packRecorder, packRequest)

	if packRecorder.Code != http.StatusOK {
		t.Fatalf("expected pack status %d, got %d body=%s", http.StatusOK, packRecorder.Code, packRecorder.Body.String())
	}

	var packPayload struct {
		WorkspacePath  string   `json:"workspacePath"`
		BundlePath     string   `json:"bundlePath"`
		BundleBase64   string   `json:"bundleBase64"`
		PackageName    string   `json:"packageName"`
		Version        string   `json:"version"`
		ManifestDigest string   `json:"manifestDigest"`
		Components     []string `json:"components"`
		Errors         []string `json:"errors"`
	}
	if err := json.Unmarshal(packRecorder.Body.Bytes(), &packPayload); err != nil {
		t.Fatalf("decode pack response: %v", err)
	}
	if packPayload.BundlePath == "" {
		t.Fatal("expected bundlePath to be returned")
	}
	if packPayload.BundleBase64 == "" {
		t.Fatal("expected bundleBase64 to be returned")
	}
	content, err := os.ReadFile(packPayload.BundlePath)
	if err != nil {
		t.Fatalf("read packed bundle: %v", err)
	}
	scanResult, validationError := artifactadapter.ScanNPMPackageTarballForTest(bytes.NewReader(content), "@platform.skill/fixture-skill", "0.2.0")
	if validationError != "" {
		t.Fatalf("expected packed bundle to validate, got %s", validationError)
	}
	if scanResult.DisplayName != "Fixture Skill" {
		t.Fatalf("unexpected packed display name: %s", scanResult.DisplayName)
	}
	packagedManifest := readTarballFile(t, content, "package/package.json")
	var packagedPackageJSON struct {
		Skill struct {
			DisplayName string `json:"displayName"`
			SourceType  string `json:"sourceType"`
		} `json:"skill"`
	}
	if err := json.Unmarshal(packagedManifest, &packagedPackageJSON); err != nil {
		t.Fatalf("decode packed package.json: %v", err)
	}
	if packagedPackageJSON.Skill.SourceType != "imported" {
		t.Fatalf("expected packed sourceType imported, got %q", packagedPackageJSON.Skill.SourceType)
	}
	if packagedPackageJSON.Skill.DisplayName != "Fixture Skill" {
		t.Fatalf("expected packed displayName Fixture Skill, got %q", packagedPackageJSON.Skill.DisplayName)
	}
}

func TestSkillPackagingPublishPublishesAndCreatesReleaseRecord(t *testing.T) {
	registry := newTestNPMRegistry(t)
	registryBaseURL := registry.baseURL()
	registryClient := registry.client()

	packagingService := packaging.NewService("")
	workspacePath, _ := prepareManagedWorkspaceForTest(
		t,
		packagingService,
		"@platform.skill/fixture-skill",
		"0.2.0",
	)

	catalogServer := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData(), catalog.MemoryRepositoryOptions{
			ArtifactSource: artifactadapter.NewNPMRegistryArtifactSource(registryBaseURL, registryClient),
		}),
		Packaging:                packagingService,
		NPMRegistryURL:           registryBaseURL,
		NPMRegistryPublishClient: registryClient,
	})

	publishRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/publish", `{
		"workspacePath":"`+workspacePath+`",
		"target":{"packageName":"@platform.skill/fixture-skill","version":"0.2.0"}
	}`)
	publishRequest.Header.Set("X-Role", "publisher")
	publishRecorder := httptest.NewRecorder()
	catalogServer.Handler().ServeHTTP(publishRecorder, publishRequest)

	if publishRecorder.Code != http.StatusCreated {
		t.Fatalf("expected publish status %d, got %d body=%s", http.StatusCreated, publishRecorder.Code, publishRecorder.Body.String())
	}

	var publishPayload struct {
		WorkspacePath       string                `json:"workspacePath"`
		PackageName         string                `json:"packageName"`
		Version             string                `json:"version"`
		ManifestDigest      string                `json:"manifestDigest"`
		Components          []string              `json:"components"`
		PublishedToRegistry bool                  `json:"publishedToRegistry"`
		Created             bool                  `json:"created"`
		IdempotentReplay    bool                  `json:"idempotentReplay"`
		VersionStatus       catalog.VersionStatus `json:"versionStatus"`
		InstallPolicy       catalog.InstallPolicy `json:"installPolicy"`
		Errors              []string              `json:"errors"`
	}
	if err := json.Unmarshal(publishRecorder.Body.Bytes(), &publishPayload); err != nil {
		t.Fatalf("decode publish response: %v", err)
	}
	if publishPayload.WorkspacePath != workspacePath {
		t.Fatalf("expected workspacePath %q, got %q", workspacePath, publishPayload.WorkspacePath)
	}
	if !publishPayload.PublishedToRegistry || !publishPayload.Created || publishPayload.IdempotentReplay {
		t.Fatalf("unexpected publish result: %+v", publishPayload)
	}
	if publishPayload.PackageName != "@platform.skill/fixture-skill" || publishPayload.Version != "0.2.0" {
		t.Fatalf("unexpected package identity: %+v", publishPayload)
	}
	if publishPayload.ManifestDigest == "" || len(publishPayload.Components) == 0 {
		t.Fatalf("expected manifest digest and components, got %+v", publishPayload)
	}
	if publishPayload.VersionStatus != catalog.VersionStatusPublished || publishPayload.InstallPolicy != catalog.InstallPolicyResolvable {
		t.Fatalf("unexpected release state: %+v", publishPayload)
	}

	replayRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/publish", `{
		"workspacePath":"`+workspacePath+`",
		"target":{"packageName":"@platform.skill/fixture-skill","version":"0.2.0"}
	}`)
	replayRequest.Header.Set("X-Role", "publisher")
	replayRecorder := httptest.NewRecorder()
	catalogServer.Handler().ServeHTTP(replayRecorder, replayRequest)

	if replayRecorder.Code != http.StatusOK {
		t.Fatalf("expected replay status %d, got %d body=%s", http.StatusOK, replayRecorder.Code, replayRecorder.Body.String())
	}

	var replayPayload struct {
		PublishedToRegistry bool `json:"publishedToRegistry"`
		Created             bool `json:"created"`
		IdempotentReplay    bool `json:"idempotentReplay"`
	}
	if err := json.Unmarshal(replayRecorder.Body.Bytes(), &replayPayload); err != nil {
		t.Fatalf("decode replay response: %v", err)
	}
	if replayPayload.PublishedToRegistry || replayPayload.Created || !replayPayload.IdempotentReplay {
		t.Fatalf("unexpected replay result: %+v", replayPayload)
	}

	searchRequest := httptest.NewRequest(http.MethodGet, "/api/v1/skills/search?q=fixture", nil)
	searchRequest.Header.Set("X-Role", "reader")
	searchRecorder := httptest.NewRecorder()
	catalogServer.Handler().ServeHTTP(searchRecorder, searchRequest)
	if searchRecorder.Code != http.StatusOK {
		t.Fatalf("expected search status %d, got %d", http.StatusOK, searchRecorder.Code)
	}
	var searchPayload struct {
		Items []catalog.SkillSummary `json:"items"`
	}
	if err := json.Unmarshal(searchRecorder.Body.Bytes(), &searchPayload); err != nil {
		t.Fatalf("decode search response: %v", err)
	}
	if len(searchPayload.Items) != 1 || searchPayload.Items[0].PackageName != "@platform.skill/fixture-skill" {
		t.Fatalf("unexpected search payload: %+v", searchPayload)
	}
}

func TestPackagingPrepareRejectsInvalidJSON(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", strings.NewReader("{"))
	request.Header.Set("X-Role", "authoring")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
}

func TestPackagingPrepareRejectsGetMethod(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skill-packaging/prepare", nil)
	request.Header.Set("X-Role", "authoring")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusMethodNotAllowed, recorder.Code, recorder.Body.String())
	}
}

func TestSkillPackagingPackMetadataAndBundleDownload(t *testing.T) {
	server := newTestServer()
	fixtureDir := createPackagingFixture(t)

	prepareRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":`+uploadArchiveSourceJSON(t, fixtureDir)+`,
		"target":{"packageName":"@platform.skill/fixture-skill","version":"0.2.0"},
		"metadata":{"displayName":"Fixture Skill","description":"Fixture packaging flow","sourceType":"imported"}
	}`)
	prepareRequest.Header.Set("X-Role", "publisher")
	prepareRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(prepareRecorder, prepareRequest)
	if prepareRecorder.Code != http.StatusOK {
		t.Fatalf("expected prepare status %d, got %d body=%s", http.StatusOK, prepareRecorder.Code, prepareRecorder.Body.String())
	}

	var preparePayload struct {
		WorkspacePath string `json:"workspacePath"`
	}
	if err := json.Unmarshal(prepareRecorder.Body.Bytes(), &preparePayload); err != nil {
		t.Fatalf("decode prepare response: %v", err)
	}

	packRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/pack", `{
		"workspacePath":"`+preparePayload.WorkspacePath+`",
		"target":{"packageName":"@platform.skill/fixture-skill","version":"0.2.0"},
		"includeBundle":false
	}`)
	packRequest.Header.Set("X-Role", "publisher")
	packRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(packRecorder, packRequest)
	if packRecorder.Code != http.StatusOK {
		t.Fatalf("expected pack status %d, got %d body=%s", http.StatusOK, packRecorder.Code, packRecorder.Body.String())
	}

	var packPayload struct {
		BundleBase64 string `json:"bundleBase64"`
	}
	if err := json.Unmarshal(packRecorder.Body.Bytes(), &packPayload); err != nil {
		t.Fatalf("decode pack response: %v", err)
	}
	if packPayload.BundleBase64 != "" {
		t.Fatalf("expected metadata-only pack response, got bundleBase64=%q", packPayload.BundleBase64)
	}

	downloadRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/skill-packaging/bundle?workspacePath="+url.QueryEscape(preparePayload.WorkspacePath)+"&packageName="+url.QueryEscape("@platform.skill/fixture-skill")+"&version=0.2.0",
		nil,
	)
	downloadRequest.Header.Set("X-Role", "publisher")
	downloadRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(downloadRecorder, downloadRequest)

	if downloadRecorder.Code != http.StatusOK {
		t.Fatalf("expected download status %d, got %d body=%s", http.StatusOK, downloadRecorder.Code, downloadRecorder.Body.String())
	}
	scanResult, validationError := artifactadapter.ScanNPMPackageTarballForTest(bytes.NewReader(downloadRecorder.Body.Bytes()), "@platform.skill/fixture-skill", "0.2.0")
	if validationError != "" {
		t.Fatalf("expected downloaded bundle to validate, got %s", validationError)
	}
	if scanResult.DisplayName != "Fixture Skill" {
		t.Fatalf("unexpected downloaded packed display name: %s", scanResult.DisplayName)
	}
}

func TestSkillPackagingSupportsPlatformAuthoringSource(t *testing.T) {
	server := newTestServer()

	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":{
			"type":"platform_authoring",
			"files":{
				"commands/run.txt":"echo authored"
			}
		},
		"target":{"packageName":"@platform.skill/platform-authored","version":"0.3.0"},
		"metadata":{"displayName":"Platform Authored","description":"Created from structured payload","sourceType":"platform_created"},
		"options":{"publishRegistry":"https://registry.example.com"}
	}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var payload struct {
		WorkspacePath string `json:"workspacePath"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(payload.WorkspacePath, "SKILL.md"))
	if err != nil {
		t.Fatalf("read generated SKILL.md: %v", err)
	}
	if !strings.Contains(string(content), "Platform Authored") {
		t.Fatalf("expected generated SKILL.md to contain display name, got %s", string(content))
	}
}

func TestSkillPackagingSupportsLocalGitSourcePath(t *testing.T) {
	server := newTestServer()
	fixtureDir := createPackagingFixture(t)

	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":{"type":"git","location":"`+fixtureDir+`"},
		"target":{"packageName":"@platform.skill/git-fixture","version":"0.4.0"},
		"metadata":{"displayName":"Git Fixture","description":"Copied from git source path","sourceType":"imported"},
		"options":{"publishRegistry":"https://registry.example.com"}
	}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var payload struct {
		WorkspacePath string `json:"workspacePath"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, err := os.Stat(filepath.Join(payload.WorkspacePath, "commands", "run.txt")); err != nil {
		t.Fatalf("expected copied command file: %v", err)
	}
}

func TestSkillPackagingSupportsRemoteGitSourceRefAndPath(t *testing.T) {
	server := newTestServer()
	remoteURL := createPackagingGitRemote(t)

	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":{"type":"git","location":"`+remoteURL+`?path=skills/pdf","ref":"release"},
		"target":{"packageName":"@platform.skill/remote-git-fixture","version":"0.5.0"},
		"metadata":{"displayName":"Remote Git Fixture","description":"Cloned from remote git source","sourceType":"imported"},
		"options":{"publishRegistry":"https://registry.example.com"}
	}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var payload struct {
		WorkspacePath string `json:"workspacePath"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	commandContent, err := os.ReadFile(filepath.Join(payload.WorkspacePath, "commands", "run.txt"))
	if err != nil {
		t.Fatalf("read copied command: %v", err)
	}
	if strings.TrimSpace(string(commandContent)) != "echo release" {
		t.Fatalf("expected release branch content, got %q", string(commandContent))
	}
	if _, err := os.Stat(filepath.Join(payload.WorkspacePath, "SKILL.md")); err != nil {
		t.Fatalf("expected SKILL.md from nested path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(payload.WorkspacePath, ".git")); !os.IsNotExist(err) {
		t.Fatalf("expected workspace to contain only packaged subtree, err=%v", err)
	}
}

func TestSkillPackagingAllowsAuthoringRole(t *testing.T) {
	server := newTestServer()
	fixtureDir := createPackagingFixture(t)

	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":`+uploadArchiveSourceJSON(t, fixtureDir)+`,
		"target":{"packageName":"@platform.skill/authoring-fixture","version":"0.6.0"}
	}`)
	request.Header.Set("X-Role", "authoring")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
}

func TestSkillPackagingPrepareNormalizesImportedLocalSkill(t *testing.T) {
	server := newTestServer()
	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte(`---
name: zoom-out
description: Broader context
---

Prompt body.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":`+uploadArchiveSourceJSON(t, sourceDir)+`,
		"target":{"packageName":"@platform.skill/zoom-out","version":"1.0.0"},
		"metadata":{"displayName":"Zoom Out","description":"Broader context","sourceType":"imported"}
	}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected prepare success, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		WorkspacePath  string   `json:"workspacePath"`
		Warnings       []string `json:"warnings"`
		WarningDetails []struct {
			Code string `json:"code"`
		} `json:"warningDetails"`
		ImportReport struct {
			Normalized                     bool     `json:"normalized"`
			GeneratedPackageJSON           bool     `json:"generatedPackageJson"`
			GeneratedCapabilityPlaceholder bool     `json:"generatedCapabilityPlaceholder"`
			GeneratedPaths                 []string `json:"generatedPaths"`
		} `json:"importReport"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	wantWarnings := []string{"normalized_missing_package_json"}
	if !reflect.DeepEqual(wantWarnings, payload.Warnings) {
		t.Fatalf("unexpected warnings (-want +got):\nwant: %#v\ngot:  %#v", wantWarnings, payload.Warnings)
	}
	if len(payload.WarningDetails) != 1 || payload.WarningDetails[0].Code != "normalized_missing_package_json" {
		t.Fatalf("unexpected warningDetails: %#v", payload.WarningDetails)
	}
	if !payload.ImportReport.Normalized || !payload.ImportReport.GeneratedPackageJSON || payload.ImportReport.GeneratedCapabilityPlaceholder {
		t.Fatalf("unexpected importReport: %#v", payload.ImportReport)
	}
}

func TestSkillPackagingPrepareStrictModePreservesMissingCapabilityFailure(t *testing.T) {
	server := newTestServer()
	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte(`---
name: zoom-out
description: Broader context
---

Prompt body.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	prepareRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":`+uploadArchiveSourceJSON(t, sourceDir)+`,
		"target":{"packageName":"@platform.skill/zoom-out","version":"1.0.0"},
		"metadata":{"displayName":"Zoom Out","description":"Broader context","sourceType":"imported"},
		"options":{"strict":true}
	}`)
	prepareRequest.Header.Set("X-Role", "publisher")
	prepareRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(prepareRecorder, prepareRequest)

	if prepareRecorder.Code != http.StatusOK {
		t.Fatalf("expected prepare success, got %d body=%s", prepareRecorder.Code, prepareRecorder.Body.String())
	}

	var preparePayload struct {
		WorkspacePath string   `json:"workspacePath"`
		Warnings      []string `json:"warnings"`
	}
	if err := json.Unmarshal(prepareRecorder.Body.Bytes(), &preparePayload); err != nil {
		t.Fatalf("decode prepare response: %v", err)
	}
	if len(preparePayload.Warnings) != 0 {
		t.Fatalf("expected no normalization warnings in strict mode, got %#v", preparePayload.Warnings)
	}

	validateRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/validate", `{
		"workspacePath":"`+preparePayload.WorkspacePath+`",
		"target":{"packageName":"@platform.skill/zoom-out","version":"1.0.0"}
	}`)
	validateRequest.Header.Set("X-Role", "publisher")
	validateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(validateRecorder, validateRequest)

	if validateRecorder.Code != http.StatusOK {
		t.Fatalf("expected validate response, got %d body=%s", validateRecorder.Code, validateRecorder.Body.String())
	}

	var validatePayload struct {
		OK     bool     `json:"ok"`
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal(validateRecorder.Body.Bytes(), &validatePayload); err != nil {
		t.Fatalf("decode validate response: %v", err)
	}
	if !validatePayload.OK {
		t.Fatalf("expected validate to succeed in strict mode, got errors %#v", validatePayload.Errors)
	}
	if len(validatePayload.Errors) != 0 {
		t.Fatalf("expected no validate errors in strict mode, got %#v", validatePayload.Errors)
	}
}

func TestSkillPackagingPrepareReturnsIdentityRewriteWarningDetails(t *testing.T) {
	server := newTestServer()
	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "SKILL.md"), []byte(`---
name: zoom-out
description: Broader context
---

Prompt body.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(sourceDir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "docs", "README.md"), []byte("# docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "package.json"), []byte(`{"name":"@upstream.skill/zoom-out","version":"9.9.9"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":`+uploadArchiveSourceJSON(t, sourceDir)+`,
		"target":{"packageName":"@platform.skill/zoom-out","version":"1.0.0"},
		"metadata":{"displayName":"Zoom Out","description":"Broader context","sourceType":"imported"},
		"options":{"strict":true}
	}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected prepare success, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Warnings       []string `json:"warnings"`
		WarningDetails []struct {
			Code    string `json:"code"`
			Title   string `json:"title"`
			Message string `json:"message"`
		} `json:"warningDetails"`
		ImportReport struct {
			Normalized             bool `json:"normalized"`
			RewrotePackageIdentity bool `json:"rewrotePackageIdentity"`
		} `json:"importReport"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	wantWarnings := []string{"normalized_rewrote_package_json_identity"}
	if !reflect.DeepEqual(wantWarnings, payload.Warnings) {
		t.Fatalf("unexpected warnings (-want +got):\nwant: %#v\ngot:  %#v", wantWarnings, payload.Warnings)
	}
	if len(payload.WarningDetails) != 1 || payload.WarningDetails[0].Code != "normalized_rewrote_package_json_identity" {
		t.Fatalf("unexpected warningDetails: %#v", payload.WarningDetails)
	}
	if !payload.ImportReport.Normalized || !payload.ImportReport.RewrotePackageIdentity {
		t.Fatalf("unexpected importReport: %#v", payload.ImportReport)
	}
}

func TestSkillPackagingRejectsEscapingPlatformAuthoringPaths(t *testing.T) {
	server := newTestServer()

	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":{
			"type":"platform_authoring",
			"files":{
				"../commands/run.txt":"echo bad"
			}
		},
		"target":{"packageName":"@platform.skill/unsafe-fixture","version":"0.7.0"},
		"metadata":{"displayName":"Unsafe Fixture","description":"Should fail"}
	}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "unsafe_path_entry")
}

func TestSkillPackagingRejectsUnreservedPackageScope(t *testing.T) {
	server := newTestServer()
	fixtureDir := createPackagingFixture(t)

	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":`+uploadArchiveSourceJSON(t, fixtureDir)+`,
		"target":{"packageName":"@other.scope/fixture-skill","version":"0.8.0"}
	}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_package_name")
}

func TestSkillPackagingPrepareIsIdempotentByKey(t *testing.T) {
	server := newTestServer()
	fixtureDir := createPackagingFixture(t)

	requestBody := `{
		"idempotencyKey":"prepare-same-request",
		"source":` + uploadArchiveSourceJSON(t, fixtureDir) + `,
		"target":{"packageName":"@platform.skill/idempotent-fixture","version":"0.9.0"}
	}`

	firstRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", requestBody)
	firstRequest.Header.Set("X-Role", "publisher")
	firstRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(firstRecorder, firstRequest)

	secondRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", requestBody)
	secondRequest.Header.Set("X-Role", "publisher")
	secondRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(secondRecorder, secondRequest)

	if firstRecorder.Code != http.StatusOK || secondRecorder.Code != http.StatusOK {
		t.Fatalf("expected both requests to succeed, got %d and %d", firstRecorder.Code, secondRecorder.Code)
	}

	var firstPayload struct {
		WorkspacePath string `json:"workspacePath"`
	}
	if err := json.Unmarshal(firstRecorder.Body.Bytes(), &firstPayload); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	var secondPayload struct {
		WorkspacePath string `json:"workspacePath"`
	}
	if err := json.Unmarshal(secondRecorder.Body.Bytes(), &secondPayload); err != nil {
		t.Fatalf("decode second response: %v", err)
	}
	if firstPayload.WorkspacePath != secondPayload.WorkspacePath {
		t.Fatalf("expected idempotent workspacePath, got %q and %q", firstPayload.WorkspacePath, secondPayload.WorkspacePath)
	}
}

func TestSkillPackagingPrepareRejectsIdempotencyKeyReuseWithDifferentBody(t *testing.T) {
	server := newTestServer()
	fixtureDir := createPackagingFixture(t)

	firstRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"idempotencyKey":"prepare-conflict",
		"source":`+uploadArchiveSourceJSON(t, fixtureDir)+`,
		"target":{"packageName":"@platform.skill/idempotent-fixture","version":"0.9.0"}
	}`)
	firstRequest.Header.Set("X-Role", "publisher")
	firstRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(firstRecorder, firstRequest)

	secondRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"idempotencyKey":"prepare-conflict",
		"source":`+uploadArchiveSourceJSON(t, fixtureDir)+`,
		"target":{"packageName":"@platform.skill/idempotent-fixture","version":"1.0.0"}
	}`)
	secondRequest.Header.Set("X-Role", "publisher")
	secondRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(secondRecorder, secondRequest)

	if firstRecorder.Code != http.StatusOK {
		t.Fatalf("expected first request to succeed, got %d", firstRecorder.Code)
	}
	if secondRecorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusConflict, secondRecorder.Code, secondRecorder.Body.String())
	}
	assertErrorCode(t, secondRecorder, "idempotency_key_conflict")
}

func TestSkillPackagingPrepareIdempotencyIgnoresJSONFieldOrder(t *testing.T) {
	server := newTestServer()
	fixtureDir := createPackagingFixture(t)

	firstRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"idempotencyKey":"prepare-field-order",
		"source":`+uploadArchiveSourceJSON(t, fixtureDir)+`,
		"target":{"packageName":"@platform.skill/idempotent-fixture","version":"0.9.0"}
	}`)
	firstRequest.Header.Set("X-Role", "publisher")
	firstRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(firstRecorder, firstRequest)

	secondRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"target":{"version":"0.9.0","packageName":"@platform.skill/idempotent-fixture"},
		"source":`+uploadArchiveSourceJSON(t, fixtureDir)+`,
		"idempotencyKey":"prepare-field-order"
	}`)
	secondRequest.Header.Set("X-Role", "publisher")
	secondRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(secondRecorder, secondRequest)

	if firstRecorder.Code != http.StatusOK || secondRecorder.Code != http.StatusOK {
		t.Fatalf("expected both requests to succeed, got %d and %d", firstRecorder.Code, secondRecorder.Code)
	}

	var firstPayload struct {
		WorkspacePath string `json:"workspacePath"`
	}
	if err := json.Unmarshal(firstRecorder.Body.Bytes(), &firstPayload); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	var secondPayload struct {
		WorkspacePath string `json:"workspacePath"`
	}
	if err := json.Unmarshal(secondRecorder.Body.Bytes(), &secondPayload); err != nil {
		t.Fatalf("decode second response: %v", err)
	}
	if firstPayload.WorkspacePath != secondPayload.WorkspacePath {
		t.Fatalf("expected same workspacePath for reordered JSON, got %q and %q", firstPayload.WorkspacePath, secondPayload.WorkspacePath)
	}
}

func TestSkillPackagingPrepareIdempotencyPersistsAcrossServerInstances(t *testing.T) {
	fixtureDir := createPackagingFixture(t)
	databaseURL := "file:" + filepath.Join(t.TempDir(), "idempotency.db")
	store, err := persist.NewSQLIdempotencyStore(context.Background(), databaseURL, time.Hour)
	if err != nil {
		t.Fatalf("create idempotency store: %v", err)
	}

	serverOne := NewServer(ServerConfig{
		Repository:  catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:   packaging.NewService(""),
		Idempotency: store,
	})

	requestBody := `{
		"idempotencyKey":"prepare-persisted",
		"source":` + uploadArchiveSourceJSON(t, fixtureDir) + `,
		"target":{"packageName":"@platform.skill/persisted-fixture","version":"0.9.1"}
	}`
	firstRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", requestBody)
	firstRequest.Header.Set("X-Role", "publisher")
	firstRecorder := httptest.NewRecorder()
	serverOne.Handler().ServeHTTP(firstRecorder, firstRequest)
	if firstRecorder.Code != http.StatusOK {
		t.Fatalf("expected first request to succeed, got %d", firstRecorder.Code)
	}

	storeReloaded, err := persist.NewSQLIdempotencyStore(context.Background(), databaseURL, time.Hour)
	if err != nil {
		t.Fatalf("reopen idempotency store: %v", err)
	}
	serverTwo := NewServer(ServerConfig{
		Repository:  catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:   packaging.NewService(""),
		Idempotency: storeReloaded,
	})

	secondRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", requestBody)
	secondRequest.Header.Set("X-Role", "publisher")
	secondRecorder := httptest.NewRecorder()
	serverTwo.Handler().ServeHTTP(secondRecorder, secondRequest)
	if secondRecorder.Code != http.StatusOK {
		t.Fatalf("expected second request to succeed, got %d body=%s", secondRecorder.Code, secondRecorder.Body.String())
	}

	var firstPayload struct {
		WorkspacePath string `json:"workspacePath"`
	}
	if err := json.Unmarshal(firstRecorder.Body.Bytes(), &firstPayload); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	var secondPayload struct {
		WorkspacePath string `json:"workspacePath"`
	}
	if err := json.Unmarshal(secondRecorder.Body.Bytes(), &secondPayload); err != nil {
		t.Fatalf("decode second response: %v", err)
	}
	if firstPayload.WorkspacePath != secondPayload.WorkspacePath {
		t.Fatalf("expected persistent workspacePath, got %q and %q", firstPayload.WorkspacePath, secondPayload.WorkspacePath)
	}
}

func TestSkillPackagingWorkspacePersistsAcrossServerInstances(t *testing.T) {
	fixtureDir := createPackagingFixture(t)
	databaseURL := "file:" + filepath.Join(t.TempDir(), "packaging-workspaces.db")
	workspaceStoreOne, err := persist.NewSQLWorkspaceStore(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("create workspace store one: %v", err)
	}
	workspaceStoreTwo, err := persist.NewSQLWorkspaceStore(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("create workspace store two: %v", err)
	}

	serverOne := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:  packaging.NewServiceWithWorkspaceStore("", workspaceStoreOne),
	})
	serverTwo := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:  packaging.NewServiceWithWorkspaceStore("", workspaceStoreTwo),
	})

	prepareRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":`+uploadArchiveSourceJSON(t, fixtureDir)+`,
		"target":{"packageName":"@platform.skill/shared-workspace","version":"0.9.3"},
		"metadata":{"displayName":"Shared Workspace","description":"Cross replica packaging flow","sourceType":"imported"}
	}`)
	prepareRequest.Header.Set("X-Role", "publisher")
	prepareRecorder := httptest.NewRecorder()
	serverOne.Handler().ServeHTTP(prepareRecorder, prepareRequest)
	if prepareRecorder.Code != http.StatusOK {
		t.Fatalf("expected prepare status %d, got %d body=%s", http.StatusOK, prepareRecorder.Code, prepareRecorder.Body.String())
	}

	var preparePayload struct {
		WorkspacePath string `json:"workspacePath"`
	}
	if err := json.Unmarshal(prepareRecorder.Body.Bytes(), &preparePayload); err != nil {
		t.Fatalf("decode prepare response: %v", err)
	}
	if strings.TrimSpace(preparePayload.WorkspacePath) == "" {
		t.Fatal("expected non-empty workspacePath")
	}
	if _, statErr := os.Stat(preparePayload.WorkspacePath); !os.IsNotExist(statErr) {
		t.Fatalf("expected persistent workspace handle instead of local directory path, stat err=%v", statErr)
	}

	validateRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/validate", `{
		"workspacePath":"`+preparePayload.WorkspacePath+`",
		"target":{"packageName":"@platform.skill/shared-workspace","version":"0.9.3"}
	}`)
	validateRequest.Header.Set("X-Role", "publisher")
	validateRecorder := httptest.NewRecorder()
	serverTwo.Handler().ServeHTTP(validateRecorder, validateRequest)
	if validateRecorder.Code != http.StatusOK {
		t.Fatalf("expected validate status %d, got %d body=%s", http.StatusOK, validateRecorder.Code, validateRecorder.Body.String())
	}

	var validatePayload struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(validateRecorder.Body.Bytes(), &validatePayload); err != nil {
		t.Fatalf("decode validate response: %v", err)
	}
	if !validatePayload.OK {
		t.Fatalf("expected cross-server validate to succeed, got body=%s", validateRecorder.Body.String())
	}

	packRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/pack", `{
		"workspacePath":"`+preparePayload.WorkspacePath+`",
		"target":{"packageName":"@platform.skill/shared-workspace","version":"0.9.3"},
		"includeBundle":false
	}`)
	packRequest.Header.Set("X-Role", "publisher")
	packRecorder := httptest.NewRecorder()
	serverTwo.Handler().ServeHTTP(packRecorder, packRequest)
	if packRecorder.Code != http.StatusOK {
		t.Fatalf("expected pack status %d, got %d body=%s", http.StatusOK, packRecorder.Code, packRecorder.Body.String())
	}

	var packPayload struct {
		BundlePath string `json:"bundlePath"`
	}
	if err := json.Unmarshal(packRecorder.Body.Bytes(), &packPayload); err != nil {
		t.Fatalf("decode pack response: %v", err)
	}
	if !strings.HasPrefix(packPayload.BundlePath, "workspace://") {
		t.Fatalf("expected persisted bundle handle, got %q", packPayload.BundlePath)
	}

	downloadRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/skill-packaging/bundle?workspacePath="+url.QueryEscape(preparePayload.WorkspacePath)+"&packageName="+url.QueryEscape("@platform.skill/shared-workspace")+"&version=0.9.3",
		nil,
	)
	downloadRequest.Header.Set("X-Role", "publisher")
	downloadRecorder := httptest.NewRecorder()
	serverOne.Handler().ServeHTTP(downloadRecorder, downloadRequest)
	if downloadRecorder.Code != http.StatusOK {
		t.Fatalf("expected bundle download status %d, got %d body=%s", http.StatusOK, downloadRecorder.Code, downloadRecorder.Body.String())
	}
	scanResult, validationError := artifactadapter.ScanNPMPackageTarballForTest(bytes.NewReader(downloadRecorder.Body.Bytes()), "@platform.skill/shared-workspace", "0.9.3")
	if validationError != "" {
		t.Fatalf("expected downloaded bundle to validate, got %s", validationError)
	}
	if scanResult.ManifestDigest == "" {
		t.Fatal("expected downloaded bundle to include a manifest digest")
	}

	cleanupRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/cleanup", `{
		"workspacePath":"`+preparePayload.WorkspacePath+`"
	}`)
	cleanupRequest.Header.Set("X-Role", "publisher")
	cleanupRecorder := httptest.NewRecorder()
	serverOne.Handler().ServeHTTP(cleanupRecorder, cleanupRequest)
	if cleanupRecorder.Code != http.StatusOK {
		t.Fatalf("expected cleanup status %d, got %d body=%s", http.StatusOK, cleanupRecorder.Code, cleanupRecorder.Body.String())
	}

	validateAfterCleanupRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/validate", `{
		"workspacePath":"`+preparePayload.WorkspacePath+`",
		"target":{"packageName":"@platform.skill/shared-workspace","version":"0.9.3"}
	}`)
	validateAfterCleanupRequest.Header.Set("X-Role", "publisher")
	validateAfterCleanupRecorder := httptest.NewRecorder()
	serverTwo.Handler().ServeHTTP(validateAfterCleanupRecorder, validateAfterCleanupRequest)
	if validateAfterCleanupRecorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected post-cleanup validate status %d, got %d body=%s", http.StatusUnprocessableEntity, validateAfterCleanupRecorder.Code, validateAfterCleanupRecorder.Body.String())
	}
	assertErrorCode(t, validateAfterCleanupRecorder, "invalid_request")
}

func TestSkillPackagingPackWithInlineBundleSkipsIdempotencyPersistence(t *testing.T) {
	fixtureDir := createPackagingFixture(t)
	databaseURL := "file:" + filepath.Join(t.TempDir(), "idempotency.db")
	store, err := persist.NewSQLIdempotencyStore(context.Background(), databaseURL, time.Hour)
	if err != nil {
		t.Fatalf("create idempotency store: %v", err)
	}

	server := NewServer(ServerConfig{
		Repository:  catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:   packaging.NewService(""),
		Idempotency: store,
	})

	prepareRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":`+uploadArchiveSourceJSON(t, fixtureDir)+`,
		"target":{"packageName":"@platform.skill/inline-bundle","version":"0.9.2"}
	}`)
	prepareRequest.Header.Set("X-Role", "publisher")
	prepareRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(prepareRecorder, prepareRequest)
	if prepareRecorder.Code != http.StatusOK {
		t.Fatalf("expected prepare status %d, got %d body=%s", http.StatusOK, prepareRecorder.Code, prepareRecorder.Body.String())
	}

	var preparePayload struct {
		WorkspacePath string `json:"workspacePath"`
	}
	if err := json.Unmarshal(prepareRecorder.Body.Bytes(), &preparePayload); err != nil {
		t.Fatalf("decode prepare response: %v", err)
	}

	packRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/pack", `{
		"idempotencyKey":"inline-bundle-pack",
		"workspacePath":"`+preparePayload.WorkspacePath+`",
		"target":{"packageName":"@platform.skill/inline-bundle","version":"0.9.2"},
		"includeBundle":true
	}`)
	packRequest.Header.Set("X-Role", "publisher")
	packRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(packRecorder, packRequest)
	if packRecorder.Code != http.StatusOK {
		t.Fatalf("expected pack status %d, got %d body=%s", http.StatusOK, packRecorder.Code, packRecorder.Body.String())
	}

	if _, err := store.Lookup(context.Background(), "pack", "publisher", "inline-bundle-pack"); !errors.Is(err, persist.ErrIdempotencyRecordNotFound) {
		t.Fatalf("expected no persisted idempotency record for inline bundle response, got err=%v", err)
	}

	if err := os.RemoveAll(preparePayload.WorkspacePath); err != nil {
		t.Fatalf("remove workspace: %v", err)
	}

	secondPackRecorder := httptest.NewRecorder()
	secondPackRequest := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/pack", `{
		"idempotencyKey":"inline-bundle-pack",
		"workspacePath":"`+preparePayload.WorkspacePath+`",
		"target":{"packageName":"@platform.skill/inline-bundle","version":"0.9.2"},
		"includeBundle":true
	}`)
	secondPackRequest.Header.Set("X-Role", "publisher")
	server.Handler().ServeHTTP(secondPackRecorder, secondPackRequest)

	if secondPackRecorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected uncached inline bundle pack to fail after workspace removal, got %d body=%s", secondPackRecorder.Code, secondPackRecorder.Body.String())
	}
}

func TestSkillPackagingPrepareWritesAuditEvent(t *testing.T) {
	fixtureDir := createPackagingFixture(t)
	databaseURL := "file:" + filepath.Join(t.TempDir(), "audit.db")
	auditStore, err := persist.NewSQLAuditStore(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("create audit store: %v", err)
	}

	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:  packaging.NewService(""),
		Audit:      auditStore,
		AuthTokens: map[string]auth.TokenConfig{
			"publisher-secret": {
				Role:  "publisher",
				Actor: "svc-publisher",
			},
		},
	})

	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":`+uploadArchiveSourceJSON(t, fixtureDir)+`,
		"target":{"packageName":"@platform.skill/audited-fixture","version":"0.9.0"}
	}`)
	request.Header.Set("Authorization", "Bearer publisher-secret")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	record := latestAuditRecord(t, auditStore, "packaging.prepare")
	if record.Actor != "svc-publisher" || record.Role != "publisher" {
		t.Fatalf("unexpected audit actor/role: %+v", record)
	}
	if record.PackageName != "@platform.skill/audited-fixture" || record.Version != "0.9.0" {
		t.Fatalf("unexpected audit package coordinates: %+v", record)
	}
	if record.Outcome != "succeeded" {
		t.Fatalf("expected succeeded audit outcome, got %+v", record)
	}
	if record.AuthDecision != string(observability.AuthDecisionAllowed) {
		t.Fatalf("expected allowed auth decision, got %+v", record)
	}
}

func TestSkillPackagingRequiresPublisherRole(t *testing.T) {
	server := newTestServer()
	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{"source":{"type":"local","location":"/tmp"},"target":{"packageName":"@platform.skill/test","version":"0.1.0"}}`)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}
	assertErrorCode(t, recorder, "forbidden")
}

func TestSkillPackagingPrepareRejectsUnknownJSONFields(t *testing.T) {
	server := newTestServer()
	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":{"type":"local","location":"/tmp"},
		"target":{"packageName":"@platform.skill/test","version":"0.1.0"},
		"unexpected":true
	}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorDetailContains(t, recorder, "error", `unknown field "unexpected"`)
}

func TestSkillPackagingPrepareRejectsMultipleJSONObjects(t *testing.T) {
	server := newTestServer()
	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{"source":{"type":"local","location":"/tmp"},"target":{"packageName":"@platform.skill/test","version":"0.1.0"}}{"source":{"type":"local","location":"/tmp"}}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorMessage(t, recorder, "request body must contain a single JSON object.")
}

func TestSkillPackagingPrepareRejectsRemovedLocalSourceType(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:  packaging.NewService(""),
	})
	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"source":{"type":"local","location":"/tmp"},
		"target":{"packageName":"@platform.skill/test","version":"0.1.0"}
	}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "unsupported_source_type")
	assertErrorDetailContains(t, recorder, "suggestedAction", "upload_archive")
}

func TestSkillPackagingPrepareReturnsInternalErrorWhenIdempotencyLookupFails(t *testing.T) {
	fixtureDir := createPackagingFixture(t)
	server := NewServer(ServerConfig{
		Repository:  catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:   packaging.NewService(""),
		Idempotency: stubIdempotencyStore{lookupErr: errors.New("lookup failed")},
	})
	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/prepare", `{
		"idempotencyKey":"prepare-error",
		"source":`+uploadArchiveSourceJSON(t, fixtureDir)+`,
		"target":{"packageName":"@platform.skill/test","version":"0.1.0"}
	}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusInternalServerError, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "internal_error")
}

func TestSkillPackagingPackReturnsNotFoundWithoutPackagingService(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
	})
	request := jsonRequest(http.MethodPost, "/api/v1/skill-packaging/pack", `{"workspacePath":"/tmp","target":{"packageName":"@platform.skill/test","version":"0.1.0"}}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
}

func TestPublishedPackageCanBeReleasedAndResolvedThroughCatalog(t *testing.T) {
	registry := newTestNPMRegistry(t)
	registryBaseURL := registry.baseURL()
	registryClient := registry.client()

	catalogServer := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData(), catalog.MemoryRepositoryOptions{
			ArtifactSource: artifactadapter.NewNPMRegistryArtifactSource(registryBaseURL, registryClient),
		}),
	})

	publishTarball := buildTarball(t, map[string]string{
		"package/SKILL.md":         "---\nname: Probe Skill\ndescription: Probe publish payload\n---\n",
		"package/package.json":     `{"name":"@platform.skill/probe-skill","version":"0.1.1"}`,
		"package/commands/run.txt": "echo run",
	})
	encodedTarball := base64.StdEncoding.EncodeToString(publishTarball)

	registry.publish(t, "@platform.skill/probe-skill", `{
		"_id":"@platform.skill/probe-skill",
		"name":"@platform.skill/probe-skill",
		"dist-tags":{"latest":"0.1.1"},
		"versions":{
			"0.1.1":{
				"name":"@platform.skill/probe-skill",
				"version":"0.1.1",
				"description":"Probe publish payload",
				"dist":{
					"integrity":"sha512-test",
					"shasum":"sha1-test",
					"tarball":"`+registryBaseURL+`/platform.skill/probe-skill/-/probe-skill-0.1.1.tgz"
				}
			}
		},
		"_attachments":{
			"@platform.skill/probe-skill-0.1.1.tgz":{
				"content_type":"application/octet-stream",
				"data":"`+encodedTarball+`",
				"length":`+itoa(len(publishTarball))+`
			}
		}
	}`)

	releaseRecordBody := `{
		"publishRegistry":"` + registryBaseURL + `",
		"manifestDigest":"` + manifestDigest("---\nname: Probe Skill\ndescription: Probe publish payload\n---\n") + `",
		"components":["commands"]
	}`
	releaseRecordRequest := jsonRequest(http.MethodPut, "/api/v1/skills/%40platform.skill%2Fprobe-skill/versions/0.1.1/release-record", releaseRecordBody)
	releaseRecordRequest.Header.Set("X-Role", "publisher")
	releaseRecordRecorder := httptest.NewRecorder()
	catalogServer.Handler().ServeHTTP(releaseRecordRecorder, releaseRecordRequest)
	if releaseRecordRecorder.Code != http.StatusCreated {
		t.Fatalf("expected release status %d, got %d", http.StatusCreated, releaseRecordRecorder.Code)
	}

	searchRequest := httptest.NewRequest(http.MethodGet, "/api/v1/skills/search?q=probe", nil)
	searchRequest.Header.Set("X-Role", "reader")
	searchRecorder := httptest.NewRecorder()
	catalogServer.Handler().ServeHTTP(searchRecorder, searchRequest)
	if searchRecorder.Code != http.StatusOK {
		t.Fatalf("expected search status %d, got %d", http.StatusOK, searchRecorder.Code)
	}
	var searchPayload struct {
		Items []catalog.SkillSummary `json:"items"`
	}
	if err := json.Unmarshal(searchRecorder.Body.Bytes(), &searchPayload); err != nil {
		t.Fatalf("decode search response: %v", err)
	}
	if len(searchPayload.Items) != 1 || searchPayload.Items[0].PackageName != "@platform.skill/probe-skill" {
		t.Fatalf("unexpected search payload: %+v", searchPayload)
	}

	installTargetRequest := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fprobe-skill/install-target", nil)
	installTargetRequest.Header.Set("X-Role", "reader")
	installTargetRecorder := httptest.NewRecorder()
	catalogServer.Handler().ServeHTTP(installTargetRecorder, installTargetRequest)
	if installTargetRecorder.Code != http.StatusOK {
		t.Fatalf("expected install target status %d, got %d", http.StatusOK, installTargetRecorder.Code)
	}
	var installTargetPayload catalog.InstallResolution
	if err := json.Unmarshal(installTargetRecorder.Body.Bytes(), &installTargetPayload); err != nil {
		t.Fatalf("decode install target response: %v", err)
	}
	if installTargetPayload.PackageName != "@platform.skill/probe-skill" || installTargetPayload.Version != "0.1.1" {
		t.Fatalf("unexpected install target payload: %+v", installTargetPayload)
	}
}

func TestSQLRepositoryBackedServerSupportsCatalogFlows(t *testing.T) {
	ctx := context.Background()
	databaseURL := "file:" + filepath.Join(t.TempDir(), "catalog.db")
	repository, err := catalog.NewSQLRepository(ctx, databaseURL, catalog.SampleData(), catalog.SQLRepositoryOptions{
		ArtifactSource: catalog.NewStaticArtifactSource(catalog.SampleData().Artifacts),
	})
	if err != nil {
		t.Fatalf("create sql repository: %v", err)
	}

	server := NewServer(ServerConfig{
		Repository: repository,
		Packaging:  packaging.NewService(""),
	})

	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/skills?q=pdf", nil)
	listRequest.Header.Set("X-Role", "reader")
	listRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRecorder, listRequest)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, listRecorder.Code)
	}

	governanceRequest := jsonRequest(http.MethodPost, "/api/v1/skills/%40platform.skill%2Fpdf-skill/versions/1.2.0/governance-changes", `{"action":"yank","reason":"security issue","replacementRecommendation":"1.1.0"}`)
	governanceRequest.Header.Set("X-Role", "publisher")
	governanceRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(governanceRecorder, governanceRequest)
	if governanceRecorder.Code != http.StatusCreated {
		t.Fatalf("expected governance status %d, got %d body=%s", http.StatusCreated, governanceRecorder.Code, governanceRecorder.Body.String())
	}

	verifyRequest := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fpdf-skill/install-target?version=1.2.0", nil)
	verifyRequest.Header.Set("X-Role", "reader")
	verifyRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(verifyRecorder, verifyRequest)
	if verifyRecorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, verifyRecorder.Code)
	}
	assertErrorCode(t, verifyRecorder, "version_yanked")
}

func TestListSkillsAndSearchAlias(t *testing.T) {
	server := newTestServer()

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/search?q=pdf", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Items []catalog.SkillSummary `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response.Items) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(response.Items))
	}
	if response.Items[0].PackageName != "@platform.skill/pdf-skill" {
		t.Fatalf("unexpected package: %s", response.Items[0].PackageName)
	}
}

func TestSkillsListUsesCursorTokens(t *testing.T) {
	server := newTestServer()

	firstRequest := httptest.NewRequest(http.MethodGet, "/api/v1/skills?pageSize=1", nil)
	firstRequest.Header.Set("X-Role", "reader")
	firstRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(firstRecorder, firstRequest)

	if firstRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, firstRecorder.Code, firstRecorder.Body.String())
	}

	var firstResponse struct {
		Items    []catalog.SkillSummary `json:"items"`
		PageInfo struct {
			NextCursor  *string `json:"nextCursor"`
			HasNextPage bool    `json:"hasNextPage"`
		} `json:"pageInfo"`
	}
	if err := json.Unmarshal(firstRecorder.Body.Bytes(), &firstResponse); err != nil {
		t.Fatalf("decode first response: %v", err)
	}
	if len(firstResponse.Items) != 1 {
		t.Fatalf("expected 1 skill on first page, got %d", len(firstResponse.Items))
	}
	if !firstResponse.PageInfo.HasNextPage || firstResponse.PageInfo.NextCursor == nil {
		t.Fatalf("expected next cursor on first page, got %+v", firstResponse.PageInfo)
	}
	if *firstResponse.PageInfo.NextCursor != "cursor_1" {
		t.Fatalf("expected token cursor_1, got %q", *firstResponse.PageInfo.NextCursor)
	}

	secondRequest := httptest.NewRequest(http.MethodGet, "/api/v1/skills?pageSize=1&cursor=cursor_1", nil)
	secondRequest.Header.Set("X-Role", "reader")
	secondRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(secondRecorder, secondRequest)

	if secondRecorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, secondRecorder.Code, secondRecorder.Body.String())
	}

	var secondResponse struct {
		Items    []catalog.SkillSummary `json:"items"`
		PageInfo struct {
			NextCursor  *string `json:"nextCursor"`
			HasNextPage bool    `json:"hasNextPage"`
		} `json:"pageInfo"`
	}
	if err := json.Unmarshal(secondRecorder.Body.Bytes(), &secondResponse); err != nil {
		t.Fatalf("decode second response: %v", err)
	}
	if len(secondResponse.Items) != 1 {
		t.Fatalf("expected 1 skill on second page, got %d", len(secondResponse.Items))
	}
	if secondResponse.Items[0].PackageName == firstResponse.Items[0].PackageName {
		t.Fatalf("expected second page to advance, got same package %s", secondResponse.Items[0].PackageName)
	}
	if secondResponse.PageInfo.HasNextPage || secondResponse.PageInfo.NextCursor != nil {
		t.Fatalf("expected second page to end pagination, got %+v", secondResponse.PageInfo)
	}
}

func TestSkillsListRejectsMalformedCursorToken(t *testing.T) {
	server := newTestServer()

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills?cursor=cursor_bad", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorDetail(t, recorder, "cursor", "cursor_bad")
}

func TestListCuratedSkills(t *testing.T) {
	server := newTestServer()

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/curated", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Items []catalog.CuratedSkillSummary `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response.Items) == 0 {
		t.Fatal("expected curated skills to be returned")
	}
	if len(response.Items) != 1 {
		t.Fatalf("expected exactly 1 curated skill, got %d", len(response.Items))
	}
	if response.Items[0].PackageName != "@platform.skill/code-review" {
		t.Fatalf("unexpected curated package: %s", response.Items[0].PackageName)
	}
	if response.Items[0].CurationReason == "" {
		t.Fatal("expected curated skill to include a curation reason")
	}
}

func TestListCuratedSkillsRejectsOversizedPageSize(t *testing.T) {
	server := newTestServer()

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/curated?pageSize=101", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorNumericDetail(t, recorder, "pageSize", 101)
}

func TestGetSkillDetailWithEncodedPackageName(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fpdf-skill", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response catalog.SkillDetail
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.PackageName != "@platform.skill/pdf-skill" {
		t.Fatalf("unexpected package: %s", response.PackageName)
	}
}

func TestGetSkillDetailReturnsNotFoundForUnknownSkill(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fmissing-skill", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "skill_not_found")
	assertErrorDetail(t, recorder, "packageName", "@platform.skill/missing-skill")
}

func TestVersionListDefaultsExcludeYanked(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fpdf-skill/versions", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Items []catalog.SkillVersionSummary `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response.Items) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(response.Items))
	}
	if response.Items[0].Version != "1.2.0" {
		t.Fatalf("expected latest visible version 1.2.0, got %s", response.Items[0].Version)
	}
}

func TestVersionListReturnsNotFoundForUnknownSkill(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fmissing-skill/versions", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "skill_not_found")
	assertErrorDetail(t, recorder, "packageName", "@platform.skill/missing-skill")
}

func TestGetVersionDetail(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fpdf-skill/versions/1.1.0", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response catalog.SkillVersionDetail
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.VersionStatus != catalog.VersionStatusDeprecated {
		t.Fatalf("expected deprecated version, got %s", response.VersionStatus)
	}
}

func TestGetVersionDetailReturnsNotFoundForUnknownVersion(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fpdf-skill/versions/9.9.9", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "version_not_found")
	assertErrorDetail(t, recorder, "packageName", "@platform.skill/pdf-skill")
	assertErrorDetail(t, recorder, "version", "9.9.9")
}

func TestGetVersionDetailRejectsNonSemverVersionPath(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fpdf-skill/versions/latest", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorDetail(t, recorder, "version", "latest")
}

func TestInstallTargetResolvesLatestPublishedVersion(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fpdf-skill/install-target", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response catalog.InstallResolution
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Version != "1.2.0" {
		t.Fatalf("expected resolved version 1.2.0, got %s", response.Version)
	}
}

func TestInstallTargetUsesPublicRegistryURLWhenConfigured(t *testing.T) {
	internalRegistry := "http://nexus.skill-registry.svc.cluster.local:8081/repository/skills-npm"
	publicRegistry := "https://skills.example.com/repository/skills-npm"
	server := NewServer(ServerConfig{
		Repository:           catalog.NewMemoryRepository(catalog.SampleDataWithRegistryBase(internalRegistry)),
		Packaging:            packaging.NewService(""),
		PublicNPMRegistryURL: publicRegistry,
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fpdf-skill/install-target", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response catalog.InstallResolution
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.NPMRegistry != publicRegistry {
		t.Fatalf("expected npmRegistry %q, got %q", publicRegistry, response.NPMRegistry)
	}

	wantTarballURL := publicRegistry + "/platform.skill/pdf-skill/-/pdf-skill-1.2.0.tgz"
	if response.TarballURL != wantTarballURL {
		t.Fatalf("expected tarballUrl %q, got %q", wantTarballURL, response.TarballURL)
	}
}

func TestInstallTargetAllowsExplicitDeprecatedVersion(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fpdf-skill/install-target?version=1.1.0", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response catalog.InstallResolution
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.VersionStatus != catalog.VersionStatusDeprecated {
		t.Fatalf("expected deprecated resolution, got %s", response.VersionStatus)
	}
}

func TestInstallTargetRejectsYankedVersionWithReplacementRecommendation(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fpdf-skill/install-target?version=1.3.0", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, recorder.Code)
	}

	assertErrorCode(t, recorder, "version_yanked")
	assertErrorDetail(t, recorder, "replacementRecommendation", "1.2.0")
}

func TestInstallTargetRejectsRetiredSkill(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40community%2Flegacy-notes/install-target", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, recorder.Code)
	}

	assertErrorCode(t, recorder, "skill_retired")
}

func TestGovernanceChangesCanBeFiltered(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/governance-changes?package_name=%40platform.skill%2Fpdf-skill&target_type=version", nil)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response struct {
		Items []catalog.GovernanceRecord `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response.Items) != 1 {
		t.Fatalf("expected 1 governance record, got %d", len(response.Items))
	}
	if response.Items[0].Action != catalog.GovernanceActionYank {
		t.Fatalf("expected yank action, got %s", response.Items[0].Action)
	}
}

func TestGovernanceChangesRejectsOversizedPageSize(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/governance-changes?pageSize=101", nil)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorNumericDetail(t, recorder, "pageSize", 101)
}

func TestSkillsRejectUnknownSortFilter(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills?sort=random", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorDetail(t, recorder, "sort", "random")
}

func TestSkillsRejectNonPositivePageSize(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills?pageSize=0", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorDetail(t, recorder, "pageSize", "0")
}

func TestSkillsRejectUnknownSourceTypeFilter(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills?source_type=legacy", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorDetail(t, recorder, "source_type", "legacy")
}

func TestSkillsRejectUnknownIncludeRetiredFilter(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills?include_retired=maybe", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorDetail(t, recorder, "include_retired", "maybe")
}

func TestVersionListRejectsUnknownIncludeDeprecatedFilter(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fpdf-skill/versions?include_deprecated=maybe", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorDetail(t, recorder, "include_deprecated", "maybe")
}

func TestVersionListRejectsUnknownIncludeYankedFilter(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fpdf-skill/versions?include_yanked=maybe", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorDetail(t, recorder, "include_yanked", "maybe")
}

func TestGovernanceChangesRejectUnknownTargetTypeFilter(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/governance-changes?target_type=package", nil)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorDetail(t, recorder, "target_type", "package")
}

func TestGovernanceChangesRejectNonSemverVersionFilter(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/governance-changes?version=latest", nil)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusBadRequest, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorDetail(t, recorder, "version", "latest")
}

func TestReleaseRecordRejectsMismatchedPublishHints(t *testing.T) {
	server := newTestServer()
	request := jsonRequest(
		http.MethodPut,
		"/api/v1/skills/%40platform.skill%2Fagent-notes/versions/0.1.0/release-record",
		`{
			"publishRegistry":"https://npm.other.example.com",
			"manifestDigest":"sha256:wrong",
			"components":["commands"]
		}`,
	)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, recorder.Code)
	}

	assertErrorCode(t, recorder, "invalid_request")
}

func TestReleaseRecordWithNPMRegistryArtifactSourcePersistsAndReloads(t *testing.T) {
	tempDir := t.TempDir()
	statePath := filepath.Join(tempDir, "catalog-state.json")
	skillContent := "---\nname: Agent Notes\ndescription: Capture agent workflow notes\n---\n"
	tarball := buildTarball(t, map[string]string{
		"package/SKILL.md":         skillContent,
		"package/package.json":     `{"name":"@platform.skill/agent-notes","version":"0.1.0"}`,
		"package/commands/run.txt": "echo run",
		"package/scripts/do.sh":    "echo script",
	})

	client := &http.Client{
		Transport: httpRoundTripFunc(func(request *http.Request) (*http.Response, error) {
			switch {
			case strings.HasSuffix(request.URL.Path, ".tgz"):
				return httpBinaryResponse(http.StatusOK, tarball), nil
			case strings.Contains(request.URL.EscapedPath(), "agent-notes"):
				return httpJSONResponse(http.StatusOK, `{
					"name":"@platform.skill/agent-notes",
					"description":"fallback description",
					"dist-tags":{"latest":"0.1.0"},
					"time":{"0.1.0":"2026-05-30T10:00:00Z"},
					"versions":{
						"0.1.0":{
							"name":"@platform.skill/agent-notes",
							"version":"0.1.0",
							"description":"fallback description",
							"dist":{
								"tarball":"https://registry.example.com/platform.skill/agent-notes/-/agent-notes-0.1.0.tgz",
								"integrity":"sha512-010",
								"shasum":"sha1-010"
							}
						}
					}
				}`), nil
			default:
				return httpJSONResponse(http.StatusNotFound, `{}`), nil
			}
		}),
	}

	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData(), catalog.MemoryRepositoryOptions{
			ArtifactSource: artifactadapter.NewNPMRegistryArtifactSource("https://registry.example.com", client),
			AggregateStore: catalog.NewJSONAggregateStore(statePath),
		}),
	})

	request := httptest.NewRequest(http.MethodPut, "/api/v1/skills/%40platform.skill%2Fagent-notes/versions/0.1.0/release-record", nil)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	verifyRequest := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fagent-notes", nil)
	verifyRequest.Header.Set("X-Role", "reader")
	verifyRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(verifyRecorder, verifyRequest)

	if verifyRecorder.Code != http.StatusOK {
		t.Fatalf("expected skill detail status %d, got %d", http.StatusOK, verifyRecorder.Code)
	}

	var detail catalog.SkillDetail
	if err := json.Unmarshal(verifyRecorder.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode skill detail: %v", err)
	}

	if detail.DisplayName != "Agent Notes" {
		t.Fatalf("expected display name from tarball scan, got %s", detail.DisplayName)
	}
	expectedDigest := manifestDigest(skillContent)
	reloaded := catalog.NewMemoryRepository(catalog.SampleData(), catalog.MemoryRepositoryOptions{
		ArtifactSource: catalog.NewStaticArtifactSource(nil),
		AggregateStore: catalog.NewJSONAggregateStore(statePath),
	})
	versionDetail, err := reloaded.GetVersion(request.Context(), "@platform.skill/agent-notes", "0.1.0")
	if err != nil {
		t.Fatalf("load persisted version: %v", err)
	}
	if versionDetail.Bundle.SkillManifestDigest != expectedDigest {
		t.Fatalf("expected persisted manifest digest %s, got %s", expectedDigest, versionDetail.Bundle.SkillManifestDigest)
	}
	if len(versionDetail.Bundle.Components) != 2 || versionDetail.Bundle.Components[0] != "commands" || versionDetail.Bundle.Components[1] != "scripts" {
		t.Fatalf("unexpected persisted components: %#v", versionDetail.Bundle.Components)
	}
}

func TestPublishedPackagePreservesSkillSourceTypeInReleaseRecord(t *testing.T) {
	registry := newTestNPMRegistry(t)
	registryBaseURL := registry.baseURL()
	registryClient := registry.client()

	catalogServer := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData(), catalog.MemoryRepositoryOptions{
			ArtifactSource: artifactadapter.NewNPMRegistryArtifactSource(registryBaseURL, registryClient),
		}),
	})

	publishTarball := buildTarball(t, map[string]string{
		"package/SKILL.md":         "---\nname: Native Probe\ndescription: Probe native publish payload\n---\n",
		"package/package.json":     `{"name":"@platform.skill/native-probe","version":"0.1.0","skill":{"displayName":"Native Probe","sourceType":"native"}}`,
		"package/commands/run.txt": "echo run",
	})
	encodedTarball := base64.StdEncoding.EncodeToString(publishTarball)

	registry.publish(t, "@platform.skill/native-probe", `{
		"_id":"@platform.skill/native-probe",
		"name":"@platform.skill/native-probe",
		"dist-tags":{"latest":"0.1.0"},
		"versions":{
			"0.1.0":{
				"name":"@platform.skill/native-probe",
				"version":"0.1.0",
				"description":"Probe native publish payload",
				"skill":{
					"displayName":"Native Probe",
					"sourceType":"native"
				},
				"dist":{
					"integrity":"sha512-native",
					"shasum":"sha1-native",
					"tarball":"`+registryBaseURL+`/platform.skill/native-probe/-/native-probe-0.1.0.tgz"
				}
			}
		},
		"_attachments":{
			"native-probe-0.1.0.tgz":{
				"content_type":"application/octet-stream",
				"data":"`+encodedTarball+`",
				"length":`+itoa(len(publishTarball))+`
			}
		}
	}`)

	releaseRecordRequest := jsonRequest(http.MethodPut, "/api/v1/skills/%40platform.skill%2Fnative-probe/versions/0.1.0/release-record", `{
		"publishRegistry":"`+registryBaseURL+`",
		"manifestDigest":"`+manifestDigest("---\nname: Native Probe\ndescription: Probe native publish payload\n---\n")+`",
		"components":["commands"]
	}`)
	releaseRecordRequest.Header.Set("X-Role", "publisher")
	releaseRecordRecorder := httptest.NewRecorder()
	catalogServer.Handler().ServeHTTP(releaseRecordRecorder, releaseRecordRequest)
	if releaseRecordRecorder.Code != http.StatusCreated {
		t.Fatalf("expected release status %d, got %d body=%s", http.StatusCreated, releaseRecordRecorder.Code, releaseRecordRecorder.Body.String())
	}

	skillRequest := httptest.NewRequest(http.MethodGet, "/api/v1/skills/%40platform.skill%2Fnative-probe", nil)
	skillRequest.Header.Set("X-Role", "reader")
	skillRecorder := httptest.NewRecorder()
	catalogServer.Handler().ServeHTTP(skillRecorder, skillRequest)
	if skillRecorder.Code != http.StatusOK {
		t.Fatalf("expected skill detail status %d, got %d body=%s", http.StatusOK, skillRecorder.Code, skillRecorder.Body.String())
	}

	var detail catalog.SkillDetail
	if err := json.Unmarshal(skillRecorder.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode skill detail: %v", err)
	}
	if detail.SourceType != catalog.SourceTypeNative {
		t.Fatalf("expected native source type, got %s", detail.SourceType)
	}
}

func TestReleaseRecordIsIdempotentWhenArtifactMatches(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/skills/%40platform.skill%2Fpdf-skill/versions/1.2.0/release-record", nil)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response catalog.ReleaseRecordResult
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Created || !response.IdempotentReplay {
		t.Fatalf("unexpected idempotent response: %+v", response)
	}
}

func TestReleaseRecordRejectsImmutableConflict(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/skills/%40platform.skill%2Fpdf-skill/versions/1.1.0/release-record", nil)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, recorder.Code)
	}

	assertErrorCode(t, recorder, "immutable_release_conflict")
}

func TestReleaseRecordRejectsInvalidBundle(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/skills/%40platform.skill%2Fbroken-bundle/versions/0.1.0/release-record", nil)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d", http.StatusUnprocessableEntity, recorder.Code)
	}

	assertErrorCode(t, recorder, "bundle_validation_failed")
}

func TestReleaseRecordRequiresPublisherRole(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/skills/%40platform.skill%2Fagent-notes/versions/0.1.0/release-record", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}

	assertErrorCode(t, recorder, "forbidden")
}

func TestSkillGovernanceRejectsRetiringAlreadyRetiredSkill(t *testing.T) {
	server := newTestServer()
	request := jsonRequest(http.MethodPost, "/api/v1/skills/%40community%2Flegacy-notes/governance-changes", `{"action":"retire","reason":"already retired"}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusConflict, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_state_transition")
}

func TestVersionGovernanceRejectsYankingAlreadyYankedVersion(t *testing.T) {
	server := newTestServer()
	request := jsonRequest(http.MethodPost, "/api/v1/skills/%40platform.skill%2Fpdf-skill/versions/1.3.0/governance-changes", `{"action":"yank","reason":"already yanked"}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusConflict, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_state_transition")
}

func TestVersionGovernanceRejectsNonSemverVersionPath(t *testing.T) {
	server := newTestServer()
	request := jsonRequest(http.MethodPost, "/api/v1/skills/%40platform.skill%2Fpdf-skill/versions/latest/governance-changes", `{"action":"yank","reason":"security issue"}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorDetail(t, recorder, "version", "latest")
}

func TestRequestLoggingIncludesRequestMetadata(t *testing.T) {
	var logBuffer bytes.Buffer
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:  packaging.NewService(""),
		Logger:     log.New(&logBuffer, "", 0),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	lines := strings.Split(strings.TrimSpace(logBuffer.String()), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		t.Fatal("expected request log line to be emitted")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &payload); err != nil {
		t.Fatalf("decode request log: %v", err)
	}

	if payload["event"] != "http_request" {
		t.Fatalf("unexpected log event: %+v", payload)
	}
	if payload["method"] != http.MethodGet || payload["path"] != "/api/v1/skills" {
		t.Fatalf("unexpected request metadata: %+v", payload)
	}
	if payload["status"] != float64(http.StatusOK) {
		t.Fatalf("unexpected request status: %+v", payload)
	}
	if payload["action"] != string(auth.ActionReadCatalog) {
		t.Fatalf("unexpected request action: %+v", payload)
	}
	if payload["authDecision"] != string(observability.AuthDecisionAllowed) {
		t.Fatalf("unexpected request auth decision: %+v", payload)
	}
	if payload["role"] != "reader" {
		t.Fatalf("unexpected request role: %+v", payload)
	}
	if _, ok := payload["requestId"].(string); !ok {
		t.Fatalf("expected requestId string, got %+v", payload)
	}
}

func TestRequestLoggingIncludesTraceIDAndResponseHeader(t *testing.T) {
	var logBuffer bytes.Buffer
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:  packaging.NewService(""),
		Logger:     log.New(&logBuffer, "", 0),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	request.Header.Set("X-Role", "reader")
	request.Header.Set("X-Trace-Id", "trace-123")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	if recorder.Header().Get("X-Trace-Id") != "trace-123" {
		t.Fatalf("expected X-Trace-Id response header, got %q", recorder.Header().Get("X-Trace-Id"))
	}

	lines := strings.Split(strings.TrimSpace(logBuffer.String()), "\n")
	var payload map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &payload); err != nil {
		t.Fatalf("decode request log: %v", err)
	}
	if payload["traceId"] != "trace-123" {
		t.Fatalf("expected traceId trace-123, got %+v", payload)
	}
}

func TestRequestMetadataMiddlewareGeneratesTraceIDAndLogsHeaders(t *testing.T) {
	var logBuffer bytes.Buffer
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:  packaging.NewService(""),
		Logger:     log.New(&logBuffer, "", 0),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	request.Header.Set("X-Role", "reader")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	requestID := recorder.Header().Get("X-Request-Id")
	if requestID == "" {
		t.Fatal("expected X-Request-Id response header")
	}

	traceID := recorder.Header().Get("X-Trace-Id")
	if traceID == "" {
		t.Fatal("expected X-Trace-Id response header")
	}

	wantTraceID := "trace_" + strings.TrimPrefix(requestID, "req_")
	if traceID != wantTraceID {
		t.Fatalf("expected generated trace ID %q, got %q", wantTraceID, traceID)
	}

	payload := findLoggedEvent(t, logBuffer.String(), "http_request")
	if payload["requestId"] != requestID {
		t.Fatalf("expected logged requestId %q, got %+v", requestID, payload)
	}
	if payload["traceId"] != traceID {
		t.Fatalf("expected logged traceId %q, got %+v", traceID, payload)
	}
}

func TestMetricsEndpointReportsRequests(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:  packaging.NewService(""),
	})

	okRequest := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	okRequest.Header.Set("X-Role", "reader")
	okRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(okRecorder, okRequest)
	if okRecorder.Code != http.StatusOK {
		t.Fatalf("expected skills status %d, got %d", http.StatusOK, okRecorder.Code)
	}

	unauthorizedRequest := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	unauthorizedRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(unauthorizedRecorder, unauthorizedRequest)
	if unauthorizedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status %d, got %d", http.StatusUnauthorized, unauthorizedRecorder.Code)
	}

	metricsRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(metricsRecorder, metricsRequest)
	if metricsRecorder.Code != http.StatusOK {
		t.Fatalf("expected metrics status %d, got %d body=%s", http.StatusOK, metricsRecorder.Code, metricsRecorder.Body.String())
	}
	body := metricsRecorder.Body.String()
	if !strings.Contains(body, `skill_registry_http_requests_total{method="GET",path="/api/v1/skills",status="200"} 1`) {
		t.Fatalf("expected 200 metrics counter, got %s", body)
	}
	if !strings.Contains(body, `skill_registry_http_requests_total{method="GET",path="/api/v1/skills",status="401"} 1`) {
		t.Fatalf("expected 401 metrics counter, got %s", body)
	}
	if !strings.Contains(body, `skill_registry_http_action_requests_total{action="catalog.read",auth_decision="allowed",outcome="succeeded",status="200"} 1`) {
		t.Fatalf("expected allowed action metrics counter, got %s", body)
	}
	if !strings.Contains(body, `skill_registry_http_action_requests_total{action="catalog.read",auth_decision="denied",outcome="unauthorized",status="401"} 1`) {
		t.Fatalf("expected denied action metrics counter, got %s", body)
	}
}

func TestHealthAndReadinessEndpointsReturnOK(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:  packaging.NewService(""),
	})

	for _, path := range []string{"/healthz", "/readyz"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected %s status %d, got %d body=%s", path, http.StatusOK, recorder.Code, recorder.Body.String())
		}

		var response map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode %s response: %v", path, err)
		}
		if response["status"] != "ok" && response["status"] != "ready" {
			t.Fatalf("unexpected %s payload: %+v", path, response)
		}
	}
}

func TestReadyzReturnsDegradedWhenDependenciesFail(t *testing.T) {
	server := NewServer(ServerConfig{
		Repository: pingingRepository{
			Repository: catalog.NewMemoryRepository(catalog.SampleData()),
			err:        errors.New("repository unavailable"),
		},
		Packaging: packaging.NewService(""),
		Audit: stubAuditStore{
			pingErr: errors.New("audit unavailable"),
		},
		Idempotency: stubIdempotencyStore{
			pingErr: errors.New("idempotency unavailable"),
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusServiceUnavailable, recorder.Code, recorder.Body.String())
	}

	var response struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != "degraded" {
		t.Fatalf("expected degraded status, got %+v", response)
	}
	if response.Checks["repository"] != "repository unavailable" {
		t.Fatalf("unexpected repository check: %+v", response.Checks)
	}
	if response.Checks["audit"] != "audit unavailable" {
		t.Fatalf("unexpected audit check: %+v", response.Checks)
	}
	if response.Checks["idempotency"] != "idempotency unavailable" {
		t.Fatalf("unexpected idempotency check: %+v", response.Checks)
	}
}

func TestSkillGovernanceRejectsMissingJSONBody(t *testing.T) {
	server := newTestServer()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/skills/%40platform.skill%2Fcode-review/governance-changes", nil)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorMessage(t, recorder, "request body is required.")
}

func TestSkillGovernanceRejectsUnknownJSONFields(t *testing.T) {
	server := newTestServer()
	request := jsonRequest(http.MethodPost, "/api/v1/skills/%40platform.skill%2Fcode-review/governance-changes", `{"action":"retire","reason":"cleanup","unexpected":true}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorDetailContains(t, recorder, "error", `unknown field "unexpected"`)
}

func TestSkillGovernanceRejectsMultipleJSONObjects(t *testing.T) {
	server := newTestServer()
	request := jsonRequest(http.MethodPost, "/api/v1/skills/%40platform.skill%2Fcode-review/governance-changes", `{"action":"retire","reason":"cleanup"}{"action":"retire"}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorMessage(t, recorder, "request body must contain a single JSON object.")
}

func TestDecodeOptionalReleaseRecordInputAllowsNilBody(t *testing.T) {
	request := httptest.NewRequest(http.MethodPut, "/api/v1/skills/%40platform.skill%2Fagent-notes/versions/0.1.0/release-record", nil)
	request.Body = nil

	payload, err := common.DecodeOptionalReleaseRecordInput(request)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !reflect.DeepEqual(payload, catalog.ReleaseRecordInput{}) {
		t.Fatalf("expected zero-value payload, got %+v", payload)
	}
}

func TestReleaseRecordRejectsUnknownJSONFields(t *testing.T) {
	server := newTestServer()
	request := jsonRequest(http.MethodPut, "/api/v1/skills/%40platform.skill%2Fagent-notes/versions/0.1.0/release-record", `{"unexpected":true}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorDetailContains(t, recorder, "error", `unknown field "unexpected"`)
}

func TestReleaseRecordRejectsMultipleJSONObjects(t *testing.T) {
	server := newTestServer()
	request := jsonRequest(http.MethodPut, "/api/v1/skills/%40platform.skill%2Fagent-notes/versions/0.1.0/release-record", `{"publishRegistry":"https://registry.example.com"}{"publishRegistry":"https://registry.example.com"}`)
	request.Header.Set("X-Role", "publisher")
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "invalid_request")
	assertErrorMessage(t, recorder, "request body must contain a single JSON object.")
}

func TestRecordMutationAuditLogsWriteFailure(t *testing.T) {
	var logBuffer bytes.Buffer
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:  packaging.NewService(""),
		Audit: stubAuditStore{
			appendErr: errors.New("append failed"),
		},
		Logger: log.New(&logBuffer, "", 0),
	})

	request := httptest.NewRequest(http.MethodPut, "/api/v1/skills/%40platform.skill%2Fagent-notes/versions/0.1.0/release-record", nil)
	ctx := common.ContextWithRequestMetadata(request.Context(), "req_audit", "trace_audit")

	server.recordMutationAudit(
		request.WithContext(ctx),
		"release.record",
		"@platform.skill/agent-notes",
		"0.1.0",
		http.StatusCreated,
		"succeeded",
		nil,
	)

	payload := findLoggedEvent(t, logBuffer.String(), "audit_write_failed")
	if payload["action"] != "release.record" {
		t.Fatalf("unexpected action payload: %+v", payload)
	}
	if payload["error"] != "append failed" {
		t.Fatalf("unexpected error payload: %+v", payload)
	}
	if payload["requestId"] != "req_audit" || payload["traceId"] != "trace_audit" {
		t.Fatalf("unexpected request metadata payload: %+v", payload)
	}
}

func TestAuthenticationDeniedAuditLogsWriteFailure(t *testing.T) {
	var logBuffer bytes.Buffer
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:  packaging.NewService(""),
		Audit: stubAuditStore{
			appendErr: errors.New("append failed"),
		},
		Logger: log.New(&logBuffer, "", 0),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	recorder := httptest.NewRecorder()

	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	}

	payload := findLoggedEvent(t, logBuffer.String(), "audit_write_failed")
	if payload["action"] != "auth.denied" {
		t.Fatalf("unexpected action payload: %+v", payload)
	}
	if payload["error"] != "append failed" {
		t.Fatalf("unexpected error payload: %+v", payload)
	}
	if _, ok := payload["requestId"].(string); !ok || payload["requestId"] == "" {
		t.Fatalf("expected requestId in payload, got %+v", payload)
	}
	if _, ok := payload["traceId"].(string); !ok || payload["traceId"] == "" {
		t.Fatalf("expected traceId in payload, got %+v", payload)
	}
}

func TestLogAuditWriteFailureIgnoresNilError(t *testing.T) {
	var logBuffer bytes.Buffer
	server := NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:  packaging.NewService(""),
		Logger:     log.New(&logBuffer, "", 0),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/skills", nil)
	ctx := common.ContextWithRequestMetadata(request.Context(), "req_nil", "trace_nil")

	server.logAuditWriteFailure(request.WithContext(ctx), "auth.denied", nil)

	if strings.TrimSpace(logBuffer.String()) != "" {
		t.Fatalf("expected no log output, got %q", logBuffer.String())
	}
}

func newTestServer() *Server {
	return NewServer(ServerConfig{
		Repository: catalog.NewMemoryRepository(catalog.SampleData()),
		Packaging:  packaging.NewService(""),
	})
}

func jsonRequest(method string, path string, body string) *http.Request {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func assertErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, want string) {
	t.Helper()

	var payload map[string]map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if payload["error"]["code"] != want {
		t.Fatalf("expected error code %s, got %#v", want, payload["error"]["code"])
	}
}

func assertErrorDetail(t *testing.T, recorder *httptest.ResponseRecorder, key string, want string) {
	t.Helper()

	var payload struct {
		Error struct {
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if payload.Error.Details[key] != want {
		t.Fatalf("expected detail %s=%s, got %#v", key, want, payload.Error.Details[key])
	}
}

func assertErrorNumericDetail(t *testing.T, recorder *httptest.ResponseRecorder, key string, want float64) {
	t.Helper()

	var payload struct {
		Error struct {
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	got, ok := payload.Error.Details[key].(float64)
	if !ok {
		t.Fatalf("expected numeric detail %s, got %#v", key, payload.Error.Details[key])
	}
	if got != want {
		t.Fatalf("expected detail %s=%v, got %#v", key, want, payload.Error.Details[key])
	}
}

func assertErrorMessage(t *testing.T, recorder *httptest.ResponseRecorder, want string) {
	t.Helper()

	var payload struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if payload.Error.Message != want {
		t.Fatalf("expected error message %q, got %q", want, payload.Error.Message)
	}
}

func assertErrorDetailContains(t *testing.T, recorder *httptest.ResponseRecorder, key string, want string) {
	t.Helper()

	var payload struct {
		Error struct {
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	value, ok := payload.Error.Details[key].(string)
	if !ok {
		t.Fatalf("expected detail %s to be a string, got %#v", key, payload.Error.Details[key])
	}
	if !strings.Contains(value, want) {
		t.Fatalf("expected detail %s to contain %q, got %q", key, want, value)
	}
}

func findLoggedEvent(t *testing.T, raw string, event string) map[string]any {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(raw), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			t.Fatalf("decode log line %q: %v", line, err)
		}
		if payload["event"] == event {
			return payload
		}
	}
	t.Fatalf("expected log event %s in %q", event, raw)
	return nil
}

type httpRoundTripFunc func(request *http.Request) (*http.Response, error)

func (fn httpRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func httpJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func httpBinaryResponse(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/octet-stream"}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

type testNPMRegistry struct {
	mu         sync.RWMutex
	packages   map[string]testNPMRegistryPackage
	base       string
	httpClient *http.Client
}

type testNPMRegistryPackage struct {
	name        string
	description string
	distTags    map[string]string
	versions    map[string]testNPMRegistryVersion
}

type testNPMRegistryVersion struct {
	name        string
	version     string
	description string
	displayName string
	sourceType  catalog.SourceType
	integrity   string
	shasum      string
	publishedAt time.Time
	tarball     []byte
}

type testNPMRegistryPublishPayload struct {
	Name        string                                   `json:"name"`
	Description string                                   `json:"description"`
	DistTags    map[string]string                        `json:"dist-tags"`
	Versions    map[string]testNPMRegistryVersionPayload `json:"versions"`
	Attachments map[string]testNPMRegistryAttachment     `json:"_attachments"`
}

type testNPMRegistryVersionPayload struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Dist        struct {
		Integrity string `json:"integrity"`
		Shasum    string `json:"shasum"`
	} `json:"dist"`
	Skill struct {
		DisplayName string             `json:"displayName"`
		SourceType  catalog.SourceType `json:"sourceType"`
	} `json:"skill"`
}

type testNPMRegistryAttachment struct {
	Data string `json:"data"`
}

func newTestNPMRegistry(t *testing.T) *testNPMRegistry {
	t.Helper()

	registry := &testNPMRegistry{
		packages: make(map[string]testNPMRegistryPackage),
		base:     "https://registry.example.com",
	}
	registry.httpClient = &http.Client{Transport: registry}
	return registry
}

func (r *testNPMRegistry) baseURL() string {
	return r.base
}

func (r *testNPMRegistry) client() *http.Client {
	return r.httpClient
}

func (r *testNPMRegistry) RoundTrip(request *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	r.serveHTTP(recorder, request.Clone(request.Context()))
	return recorder.Result(), nil
}

func (r *testNPMRegistry) publish(t *testing.T, packageName string, body string) {
	t.Helper()

	request, err := http.NewRequest(http.MethodPut, r.baseURL()+"/"+url.PathEscape(packageName), strings.NewReader(body))
	if err != nil {
		t.Fatalf("create registry publish request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := r.client().Do(request)
	if err != nil {
		t.Fatalf("publish package to test registry: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		content, _ := io.ReadAll(response.Body)
		t.Fatalf("expected publish status %d, got %d body=%s", http.StatusCreated, response.StatusCode, strings.TrimSpace(string(content)))
	}
}

func (r *testNPMRegistry) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	remainder := strings.TrimPrefix(request.URL.EscapedPath(), "/")
	if remainder == "" {
		http.NotFound(writer, request)
		return
	}
	if strings.Contains(remainder, "/-/") {
		r.handleTarball(writer, request, remainder)
		return
	}

	packageName, err := url.PathUnescape(remainder)
	if err != nil {
		http.NotFound(writer, request)
		return
	}

	switch request.Method {
	case http.MethodGet:
		r.handleMetadata(writer, request, packageName)
	case http.MethodPut:
		r.handlePublish(writer, request, packageName)
	default:
		writer.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (r *testNPMRegistry) handleMetadata(writer http.ResponseWriter, request *http.Request, packageName string) {
	r.mu.RLock()
	pkg, ok := r.packages[packageName]
	r.mu.RUnlock()
	if !ok {
		http.NotFound(writer, request)
		return
	}

	versions := make(map[string]map[string]any, len(pkg.versions))
	timeMap := make(map[string]string, len(pkg.versions))
	for version, item := range pkg.versions {
		versionPayload := map[string]any{
			"name":        item.name,
			"version":     item.version,
			"description": item.description,
			"dist": map[string]any{
				"tarball":   r.tarballURL(packageName, version),
				"integrity": item.integrity,
				"shasum":    item.shasum,
			},
		}
		if item.displayName != "" || item.sourceType != "" {
			versionPayload["skill"] = map[string]any{
				"displayName": item.displayName,
				"sourceType":  item.sourceType,
			}
		}
		versions[version] = versionPayload
		timeMap[version] = item.publishedAt.UTC().Format(time.RFC3339)
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(writer).Encode(map[string]any{
		"name":        pkg.name,
		"description": pkg.description,
		"dist-tags":   pkg.distTags,
		"time":        timeMap,
		"versions":    versions,
	}); err != nil {
		t := strings.TrimSpace(err.Error())
		http.Error(writer, t, http.StatusInternalServerError)
	}
}

func (r *testNPMRegistry) handlePublish(writer http.ResponseWriter, request *http.Request, packageName string) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(writer, "failed to read request body", http.StatusBadRequest)
		return
	}

	var payload testNPMRegistryPublishPayload
	if err := json.Unmarshal(body, &payload); err != nil || payload.Name != packageName {
		http.Error(writer, "invalid publish payload", http.StatusBadRequest)
		return
	}

	published := testNPMRegistryPackage{
		name:        payload.Name,
		description: strings.TrimSpace(payload.Description),
		distTags:    payload.DistTags,
		versions:    make(map[string]testNPMRegistryVersion, len(payload.Versions)),
	}
	for version, item := range payload.Versions {
		attachment, ok := payload.Attachments[packageName+"-"+version+".tgz"]
		if !ok {
			attachment, ok = payload.Attachments[testNPMRegistryBasename(packageName)+"-"+version+".tgz"]
		}
		if !ok {
			attachment, ok = payload.Attachments[testNPMRegistryAttachmentName(packageName, version)]
		}
		if !ok {
			continue
		}
		tarball, err := base64.StdEncoding.DecodeString(attachment.Data)
		if err != nil {
			http.Error(writer, "invalid publish payload", http.StatusBadRequest)
			return
		}
		published.versions[version] = testNPMRegistryVersion{
			name:        item.Name,
			version:     item.Version,
			description: item.Description,
			displayName: item.Skill.DisplayName,
			sourceType:  item.Skill.SourceType,
			integrity:   item.Dist.Integrity,
			shasum:      item.Dist.Shasum,
			publishedAt: time.Now().UTC(),
			tarball:     tarball,
		}
		if published.description == "" {
			published.description = strings.TrimSpace(item.Description)
		}
	}

	r.mu.Lock()
	r.packages[packageName] = published
	r.mu.Unlock()
	writer.WriteHeader(http.StatusCreated)
}

func (r *testNPMRegistry) handleTarball(writer http.ResponseWriter, request *http.Request, remainder string) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	packagePart, filePart, ok := strings.Cut(remainder, "/-/")
	if !ok {
		http.NotFound(writer, request)
		return
	}
	packageName := "@" + packagePart
	version := strings.TrimSuffix(strings.TrimPrefix(filePart, testNPMRegistryBasename(packageName)+"-"), ".tgz")

	r.mu.RLock()
	pkg, ok := r.packages[packageName]
	if !ok {
		r.mu.RUnlock()
		http.NotFound(writer, request)
		return
	}
	item, ok := pkg.versions[version]
	r.mu.RUnlock()
	if !ok {
		http.NotFound(writer, request)
		return
	}

	writer.Header().Set("Content-Type", "application/octet-stream")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(item.tarball)
}

func (r *testNPMRegistry) tarballURL(packageName string, version string) string {
	return r.baseURL() + "/" + strings.TrimPrefix(packageName, "@") + "/-/" + testNPMRegistryBasename(packageName) + "-" + version + ".tgz"
}

func testNPMRegistryAttachmentName(packageName string, version string) string {
	return testNPMRegistryBasename(packageName) + "-" + version + ".tgz"
}

func testNPMRegistryBasename(packageName string) string {
	parts := strings.Split(packageName, "/")
	return parts[len(parts)-1]
}

type pingingRepository struct {
	catalog.Repository
	err error
}

func (r pingingRepository) Ping(context.Context) error {
	return r.err
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

type stubIdempotencyStore struct {
	lookupRecord     persist.IdempotencyRecord
	lookupErr        error
	saveOK           bool
	saveErr          error
	deleteExpiredErr error
	pingErr          error
}

func (s stubIdempotencyStore) Lookup(context.Context, string, string, string) (persist.IdempotencyRecord, error) {
	return s.lookupRecord, s.lookupErr
}

func (s stubIdempotencyStore) Save(context.Context, string, string, string, persist.IdempotencyRecord) (bool, error) {
	return s.saveOK, s.saveErr
}

func (s stubIdempotencyStore) DeleteExpired(context.Context) error {
	return s.deleteExpiredErr
}

func (s stubIdempotencyStore) Ping(context.Context) error {
	return s.pingErr
}

type capturingCatalogReadWorkflow struct {
	listSkillsCalls atomic.Int64
	principal       auth.Principal
	query           catalog.SkillQuery
	response        catalog.SkillsPage
	err             *catalog.Error
}

func (w *capturingCatalogReadWorkflow) listSkills(_ context.Context, principal auth.Principal, query catalog.SkillQuery) (catalog.SkillsPage, *catalog.Error) {
	w.listSkillsCalls.Add(1)
	w.principal = principal
	w.query = query
	return w.response, w.err
}

func (w *capturingCatalogReadWorkflow) listCuratedSkills(context.Context, auth.Principal, int, int) (catalog.CuratedSkillsPage, *catalog.Error) {
	return catalog.CuratedSkillsPage{}, nil
}

func (w *capturingCatalogReadWorkflow) listSkillsCallCount() int {
	return int(w.listSkillsCalls.Load())
}

type testAuditRecord struct {
	Action       string
	Actor        string
	Role         string
	PackageName  string
	Version      string
	Outcome      string
	AuthDecision string
}

func latestAuditRecord(t *testing.T, store *persist.SQLAuditStore, action string) testAuditRecord {
	t.Helper()

	page, err := store.List(context.Background(), persist.AuditQuery{
		Action:   action,
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("load audit record: %v", err)
	}
	if len(page.Items) == 0 {
		t.Fatalf("expected audit record for action %s", action)
	}
	record := testAuditRecord{
		Action:       page.Items[0].Action,
		Actor:        page.Items[0].Actor,
		Role:         page.Items[0].Role,
		PackageName:  page.Items[0].PackageName,
		Outcome:      page.Items[0].Outcome,
		AuthDecision: string(page.Items[0].AuthDecision),
	}
	if page.Items[0].Version != nil {
		record.Version = *page.Items[0].Version
	}
	return record
}

func buildTarball(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			t.Fatalf("write tar content: %v", err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return archive.Bytes()
}

func readTarballFile(t *testing.T, tarball []byte, target string) []byte {
	t.Helper()

	gzipReader, err := gzip.NewReader(bytes.NewReader(tarball))
	if err != nil {
		t.Fatalf("open gzip reader: %v", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read tar entry: %v", err)
		}
		if header.Name != target {
			continue
		}
		content, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatalf("read tar entry content: %v", err)
		}
		return content
	}

	t.Fatalf("tarball missing file %s", target)
	return nil
}

func createPackagingFixture(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: Fixture Skill\ndescription: Fixture packaging flow\n---\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "commands"), 0o755); err != nil {
		t.Fatalf("mkdir commands: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "commands", "run.txt"), []byte("echo fixture"), 0o644); err != nil {
		t.Fatalf("write command: %v", err)
	}
	return dir
}

func uploadArchiveSourceJSON(t *testing.T, root string) string {
	t.Helper()

	payload, err := json.Marshal(packaging.Source{
		Type:          "upload_archive",
		ArchiveBase64: buildSourceArchiveBase64FromDirectory(t, root),
	})
	if err != nil {
		t.Fatalf("marshal upload archive source: %v", err)
	}
	return string(payload)
}

func buildSourceArchiveBase64FromDirectory(t *testing.T, root string) string {
	t.Helper()

	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)
		if relativePath == ".dist" || strings.HasPrefix(relativePath, ".dist/") || relativePath == ".git" || strings.HasPrefix(relativePath, ".git/") || relativePath == ".skill-packaging-import-report.json" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relativePath
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = tarWriter.Write(content)
		return err
	})
	if err != nil {
		t.Fatalf("build source archive: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return base64.StdEncoding.EncodeToString(archive.Bytes())
}

func createPackagingGitRemote(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoDir := filepath.Join(root, "repo")
	if err := os.MkdirAll(filepath.Join(repoDir, "skills", "pdf", "commands"), 0o755); err != nil {
		t.Fatalf("mkdir repo fixture: %v", err)
	}
	writeFile := func(path string, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeFile(filepath.Join(repoDir, "skills", "pdf", "SKILL.md"), "---\nname: Remote Git Fixture\ndescription: Git packaging flow\n---\n")
	writeFile(filepath.Join(repoDir, "skills", "pdf", "commands", "run.txt"), "echo main\n")

	runGit := func(dir string, args ...string) {
		t.Helper()
		command := exec.Command("git", args...)
		command.Dir = dir
		command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v output=%s", args, err, string(output))
		}
	}

	runGit(repoDir, "init")
	runGit(repoDir, "config", "user.name", "Test User")
	runGit(repoDir, "config", "user.email", "test@example.com")
	runGit(repoDir, "add", ".")
	runGit(repoDir, "commit", "-m", "main")
	runGit(repoDir, "branch", "-M", "main")
	runGit(repoDir, "checkout", "-b", "release")
	writeFile(filepath.Join(repoDir, "skills", "pdf", "commands", "run.txt"), "echo release\n")
	runGit(repoDir, "add", ".")
	runGit(repoDir, "commit", "-m", "release")

	remoteDir := filepath.Join(root, "remote.git")
	runGit(root, "clone", "--bare", repoDir, remoteDir)
	return "file://" + remoteDir
}

func manifestDigest(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func itoa(value int) string {
	return strconv.Itoa(value)
}

func mustPathFromURL(t *testing.T, raw string) string {
	t.Helper()

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url %q: %v", raw, err)
	}
	return parsed.EscapedPath()
}
