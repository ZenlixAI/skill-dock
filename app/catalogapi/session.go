package catalogapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"skill_registry/app/auth"
	"skill_registry/business/catalog"
)

type SessionResponse struct {
	Role            string   `json:"role"`
	Actor           string   `json:"actor"`
	AuthMechanism   string   `json:"authMechanism"`
	AllowedActions  []string `json:"allowedActions"`
	AllowedPackages []string `json:"allowedPackages"`
}

func buildSessionResponse(role string, actor string, authMechanism string, allowedActions []string, allowedPackages []string) SessionResponse {
	packages := append([]string{}, allowedPackages...)

	return SessionResponse{
		Role:            role,
		Actor:           actor,
		AuthMechanism:   authMechanism,
		AllowedActions:  append([]string{}, allowedActions...),
		AllowedPackages: packages,
	}
}

type sessionLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type sessionLogoutResponse struct {
	OK bool `json:"ok"`
}

func (s *Server) handleSession(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		writeCatalogError(writer, request, catalog.NewError(http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is allowed.", nil))
		return
	}

	current, ok := s.authenticateRequest(request)
	if !ok {
		writeJSON(writer, http.StatusOK, nil)
		return
	}
	if authErr := s.authorizer.AuthorizeAction(current, auth.ActionReadSession); authErr != nil {
		s.accessWorkflow.deny(writer, request, current, authErr, "", "")
		return
	}

	request = request.WithContext(auth.ContextWithPrincipal(request.Context(), current))
	current = auth.PrincipalFromContext(request.Context())
	writeJSON(writer, http.StatusOK, s.sessionModule.current(current))
}

func (s *Server) handleSessionLogin(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost)
		writeCatalogError(writer, request, catalog.NewError(http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed.", nil))
		return
	}
	if s.sessionModule == nil || !s.sessionModule.enabled() {
		writeCatalogError(writer, request, catalog.NewError(http.StatusNotFound, "session_login_disabled", "Session login is not enabled for this deployment.", nil))
		return
	}

	var input sessionLoginRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeCatalogError(writer, request, catalog.NewError(http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.", nil))
		return
	}
	if strings.TrimSpace(input.Username) == "" || strings.TrimSpace(input.Password) == "" {
		writeCatalogError(writer, request, catalog.NewError(http.StatusBadRequest, "invalid_request", "username and password are required.", nil))
		return
	}

	current, cookie, ok := s.sessionModule.login(request, input.Username, input.Password)
	if !ok {
		writeCatalogError(writer, request, catalog.NewError(http.StatusUnauthorized, "unauthorized", "Invalid username or password.", nil))
		return
	}

	http.SetCookie(writer, cookie)
	writeJSON(writer, http.StatusOK, s.sessionModule.current(current))
}

func (s *Server) handleSessionLogout(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.Header().Set("Allow", http.MethodPost)
		writeCatalogError(writer, request, catalog.NewError(http.StatusMethodNotAllowed, "method_not_allowed", "Only POST is allowed.", nil))
		return
	}

	if s.sessionModule != nil {
		if cookie := s.sessionModule.logoutCookie(request); cookie != nil {
			http.SetCookie(writer, cookie)
		}
	}
	writeJSON(writer, http.StatusOK, sessionLogoutResponse{OK: true})
}
