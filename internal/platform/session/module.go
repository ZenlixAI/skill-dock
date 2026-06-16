package session

import (
	"net/http"

	"skill_registry/internal/platform/auth"
)

type Workflow interface {
	Current(auth.Principal) Response
	Login(request *http.Request, username string, password string) (auth.Principal, *http.Cookie, bool)
	LogoutCookie(request *http.Request) *http.Cookie
	Enabled() bool
}

type Module struct {
	authorizer     *auth.Authorizer
	sessionManager *auth.SessionManager
}

func NewModule(authorizer *auth.Authorizer, sessionManager *auth.SessionManager) *Module {
	return &Module{
		authorizer:     authorizer,
		sessionManager: sessionManager,
	}
}

func (m *Module) Current(principal auth.Principal) Response {
	var allowedActions []string
	if m.authorizer != nil {
		allowedActions = m.authorizer.AllowedActionsForRole(principal.Role)
	}

	return buildResponse(
		principal.Role,
		principal.Actor,
		principal.AuthMechanism,
		allowedActions,
		principal.AllowedPackages,
	)
}

func (m *Module) Login(request *http.Request, username string, password string) (auth.Principal, *http.Cookie, bool) {
	if m.sessionManager == nil {
		return auth.Principal{}, nil, false
	}
	return m.sessionManager.Login(request, username, password)
}

func (m *Module) LogoutCookie(request *http.Request) *http.Cookie {
	if m.sessionManager == nil || !m.sessionManager.Enabled() {
		return nil
	}
	return m.sessionManager.LogoutCookie(request)
}

func (m *Module) Enabled() bool {
	return m.sessionManager != nil && m.sessionManager.Enabled()
}
