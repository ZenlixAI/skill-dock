package catalogapi

import (
	"net/http"

	"skill_registry/app/auth"
)

type sessionResponse = SessionResponse

type sessionWorkflow interface {
	current(principal auth.Principal) sessionResponse
	login(request *http.Request, username string, password string) (auth.Principal, *http.Cookie, bool)
	logoutCookie(request *http.Request) *http.Cookie
	enabled() bool
}

type sessionModule struct {
	authorizer     *auth.Authorizer
	sessionManager *auth.SessionManager
}

func (m *sessionModule) current(principal auth.Principal) sessionResponse {
	return buildSessionResponse(
		principal.Role,
		principal.Actor,
		principal.AuthMechanism,
		m.authorizer.AllowedActionsForRole(principal.Role),
		principal.AllowedPackages,
	)
}

func (m *sessionModule) login(request *http.Request, username string, password string) (auth.Principal, *http.Cookie, bool) {
	if m.sessionManager == nil {
		return auth.Principal{}, nil, false
	}
	return m.sessionManager.Login(request, username, password)
}

func (m *sessionModule) logoutCookie(request *http.Request) *http.Cookie {
	if m.sessionManager == nil || !m.sessionManager.Enabled() {
		return nil
	}
	return m.sessionManager.LogoutCookie(request)
}

func (m *sessionModule) enabled() bool {
	return m.sessionManager != nil && m.sessionManager.Enabled()
}
