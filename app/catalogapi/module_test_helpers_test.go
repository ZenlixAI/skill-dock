package catalogapi

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"skill_registry/app/auth"
	"skill_registry/business/catalog"
	"skill_registry/business/packaging"
)

type catalogReadModuleStub struct {
	listSkillsCalls atomic.Int64
}

func newCatalogReadModuleStubForTest(_ interface{ Helper() }) *catalogReadModuleStub {
	return &catalogReadModuleStub{}
}

func newTestServerWithCatalogReadModule(_ interface{ Helper() }, module catalogReadWorkflow) *Server {
	server := newTestServer()
	server.catalogReadModule = module
	return server
}

func (m *catalogReadModuleStub) listSkills(context.Context, auth.Principal, catalog.SkillQuery) (catalog.SkillsPage, *catalog.Error) {
	m.listSkillsCalls.Add(1)
	return catalog.SkillsPage{}, nil
}

func (m *catalogReadModuleStub) listCuratedSkills(context.Context, auth.Principal, int, int) (catalog.CuratedSkillsPage, *catalog.Error) {
	return catalog.CuratedSkillsPage{}, nil
}

func (m *catalogReadModuleStub) listSkillsCallCount() int {
	return int(m.listSkillsCalls.Load())
}

func newSessionModuleForTest(_ interface{ Helper() }) *sessionModule {
	return &sessionModule{
		authorizer: auth.NewAuthorizer(auth.Config{
			PublisherToken: "publisher-secret",
		}),
	}
}

type sessionModuleStub struct {
	currentCalls atomic.Int64
	loginCalls   atomic.Int64
	logoutCalls  atomic.Int64
}

func newSessionModuleStubForTest(_ interface{ Helper() }) *sessionModuleStub {
	return &sessionModuleStub{}
}

func newTestServerWithSessionModule(_ interface{ Helper() }, module sessionWorkflow) *Server {
	server := newTestServer()
	server.sessionModule = module
	return server
}

func (m *sessionModuleStub) current(auth.Principal) sessionResponse {
	m.currentCalls.Add(1)
	return sessionResponse{
		Role:            "publisher",
		Actor:           "delegated",
		AuthMechanism:   "delegated_session",
		AllowedActions:  []string{"delegated.action"},
		AllowedPackages: []string{},
	}
}

func (m *sessionModuleStub) login(_ *http.Request, _, _ string) (auth.Principal, *http.Cookie, bool) {
	m.loginCalls.Add(1)
	return auth.Principal{
		Role:          "publisher",
		Actor:         "delegated",
		AuthMechanism: "delegated_session",
	}, &http.Cookie{Name: auth.SessionCookieName, Value: "delegated-cookie", Path: "/"}, true
}

func (m *sessionModuleStub) logoutCookie(_ *http.Request) *http.Cookie {
	m.logoutCalls.Add(1)
	return &http.Cookie{Name: auth.SessionCookieName, Value: "", Path: "/", MaxAge: -1}
}

func (m *sessionModuleStub) enabled() bool {
	return true
}

func (m *sessionModuleStub) currentCallCount() int {
	return int(m.currentCalls.Load())
}

func (m *sessionModuleStub) loginCallCount() int {
	return int(m.loginCalls.Load())
}

func (m *sessionModuleStub) logoutCallCount() int {
	return int(m.logoutCalls.Load())
}

type testPackagingModule struct {
	*packagingModule
	service *stubPackagingService
}

type stubPackagingService struct {
	prepareCalls atomic.Int64
	prepareFunc  func(packaging.PrepareRequest) (packaging.PrepareResponse, error)
	validateFunc func(packaging.ValidateRequest) (packaging.ValidateResponse, error)
	packFunc     func(packaging.PackRequest) (packaging.PackResponse, error)
}

func (s *stubPackagingService) Prepare(input packaging.PrepareRequest) (packaging.PrepareResponse, error) {
	s.prepareCalls.Add(1)
	if s.prepareFunc != nil {
		return s.prepareFunc(input)
	}
	return packaging.PrepareResponse{}, nil
}

func (s *stubPackagingService) Validate(input packaging.ValidateRequest) (packaging.ValidateResponse, error) {
	if s.validateFunc != nil {
		return s.validateFunc(input)
	}
	return packaging.ValidateResponse{}, nil
}

func (s *stubPackagingService) Pack(input packaging.PackRequest) (packaging.PackResponse, error) {
	if s.packFunc != nil {
		return s.packFunc(input)
	}
	return packaging.PackResponse{}, nil
}

func newPackagingModuleForTest(_ interface{ Helper() }) *testPackagingModule {
	service := &stubPackagingService{
		prepareFunc: func(input packaging.PrepareRequest) (packaging.PrepareResponse, error) {
			return packaging.PrepareResponse{
				WorkspacePath: "cached-workspace",
				PackageName:   input.Target.PackageName,
				Version:       input.Target.Version,
				Warnings:      []string{},
				Errors:        []string{},
			}, nil
		},
	}
	return &testPackagingModule{
		packagingModule: &packagingModule{
			service:             service,
			cache:               newPackagingResponseCache(10 * time.Minute),
			allowAnonymousCache: true,
		},
		service: service,
	}
}

func (m *testPackagingModule) prepareCallCount() int {
	return int(m.service.prepareCalls.Load())
}
