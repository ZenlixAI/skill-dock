package sessionhandlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"skill_registry/app/catalogapi/response"
	catalog "skill_registry/internal/catalog/domain"
	"skill_registry/internal/platform/auth"
)

type AuthenticateFunc func(*http.Request) (auth.Principal, bool)
type AuthorizeActionFunc func(http.ResponseWriter, *http.Request, auth.Action, string, string) bool
type CurrentFunc func(auth.Principal) any
type LoginFunc func(*http.Request, string, string) (auth.Principal, *http.Cookie, bool)
type LogoutCookieFunc func(*http.Request) *http.Cookie
type EnabledFunc func() bool

type Handlers struct {
	Authenticate    AuthenticateFunc
	AuthorizeAction AuthorizeActionFunc
	CurrentFunc     CurrentFunc
	LoginFunc       LoginFunc
	LogoutCookie    LogoutCookieFunc
	Enabled         EnabledFunc
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type logoutResponse struct {
	OK bool `json:"ok"`
}

func New(config Handlers) Handlers {
	return config
}

func (h Handlers) Current(context *gin.Context) {
	current := auth.PrincipalFromContext(context.Request.Context())
	if !principalPresent(current) && h.Authenticate != nil {
		current, _ = h.Authenticate(context.Request)
	}
	if !principalPresent(current) {
		response.WriteJSON(context, http.StatusOK, nil)
		return
	}

	context.Request = context.Request.WithContext(auth.ContextWithPrincipal(context.Request.Context(), current))
	if h.AuthorizeAction != nil && !h.AuthorizeAction(context.Writer, context.Request, auth.ActionReadSession, "", "") {
		return
	}

	if h.CurrentFunc == nil {
		response.WriteJSON(context, http.StatusOK, nil)
		return
	}
	response.WriteJSON(context, http.StatusOK, h.CurrentFunc(auth.PrincipalFromContext(context.Request.Context())))
}

func (h Handlers) Login(context *gin.Context) {
	if h.Enabled == nil || !h.Enabled() {
		response.WriteCatalogError(context, catalog.NewError(http.StatusNotFound, "session_login_disabled", "Session login is not enabled for this deployment.", nil))
		return
	}

	var input loginRequest
	if err := context.ShouldBindJSON(&input); err != nil {
		response.WriteCatalogError(context, catalog.NewError(http.StatusBadRequest, "invalid_request", "Request body must be valid JSON.", nil))
		return
	}
	if strings.TrimSpace(input.Username) == "" || strings.TrimSpace(input.Password) == "" {
		response.WriteCatalogError(context, catalog.NewError(http.StatusBadRequest, "invalid_request", "username and password are required.", nil))
		return
	}

	if h.LoginFunc == nil || h.CurrentFunc == nil {
		response.WriteCatalogError(context, catalog.NewError(http.StatusUnauthorized, "unauthorized", "Invalid username or password.", nil))
		return
	}

	current, cookie, ok := h.LoginFunc(context.Request, input.Username, input.Password)
	if !ok {
		response.WriteCatalogError(context, catalog.NewError(http.StatusUnauthorized, "unauthorized", "Invalid username or password.", nil))
		return
	}

	if cookie != nil {
		http.SetCookie(context.Writer, cookie)
	}
	response.WriteJSON(context, http.StatusOK, h.CurrentFunc(current))
}

func (h Handlers) Logout(context *gin.Context) {
	if h.LogoutCookie != nil {
		if cookie := h.LogoutCookie(context.Request); cookie != nil {
			http.SetCookie(context.Writer, cookie)
		}
	}
	response.WriteJSON(context, http.StatusOK, logoutResponse{OK: true})
}

func principalPresent(current auth.Principal) bool {
	return strings.TrimSpace(current.Role) != "" ||
		strings.TrimSpace(current.Actor) != "" ||
		strings.TrimSpace(current.AuthMechanism) != "" ||
		len(current.AllowedPackages) > 0
}
