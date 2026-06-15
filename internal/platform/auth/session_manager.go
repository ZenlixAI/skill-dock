package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const SessionCookieName = "skill_registry_session"

type SessionUserConfig struct {
	Password        string   `json:"password"`
	Role            string   `json:"role"`
	Actor           string   `json:"actor,omitempty"`
	AllowedPackages []string `json:"allowedPackages,omitempty"`
}

type SessionManager struct {
	secret     []byte
	users      map[string]SessionUserConfig
	cookieName string
	ttl        time.Duration
}

type sessionCookiePayload struct {
	Username  string `json:"username"`
	ExpiresAt int64  `json:"expiresAt"`
}

func NewSessionManager(secret string, users map[string]SessionUserConfig, ttl time.Duration) *SessionManager {
	trimmedSecret := strings.TrimSpace(secret)
	if trimmedSecret == "" || len(users) == 0 {
		return nil
	}
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}

	normalizedUsers := make(map[string]SessionUserConfig, len(users))
	for username, user := range users {
		normalizedUsername := strings.TrimSpace(username)
		if normalizedUsername == "" {
			continue
		}
		role := strings.TrimSpace(user.Role)
		password := strings.TrimSpace(user.Password)
		if role == "" || password == "" {
			continue
		}
		normalizedUsers[normalizedUsername] = SessionUserConfig{
			Password:        password,
			Role:            role,
			Actor:           strings.TrimSpace(user.Actor),
			AllowedPackages: append([]string(nil), user.AllowedPackages...),
		}
	}
	if len(normalizedUsers) == 0 {
		return nil
	}

	return &SessionManager{
		secret:     []byte(trimmedSecret),
		users:      normalizedUsers,
		cookieName: SessionCookieName,
		ttl:        ttl,
	}
}

func (m *SessionManager) Enabled() bool {
	return m != nil && len(m.users) > 0 && len(m.secret) > 0
}

func (m *SessionManager) Authenticate(request *http.Request) (Principal, bool) {
	if !m.Enabled() {
		return Principal{}, false
	}

	cookie, err := request.Cookie(m.cookieName)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return Principal{}, false
	}

	payload, ok := m.decodeCookie(cookie.Value)
	if !ok {
		return Principal{}, false
	}
	if payload.ExpiresAt <= time.Now().Unix() {
		return Principal{}, false
	}

	user, ok := m.users[payload.Username]
	if !ok {
		return Principal{}, false
	}
	return sessionPrincipalFromUser(user), true
}

func (m *SessionManager) Login(request *http.Request, username string, password string) (Principal, *http.Cookie, bool) {
	if !m.Enabled() {
		return Principal{}, nil, false
	}

	user, ok := m.users[strings.TrimSpace(username)]
	if !ok {
		return Principal{}, nil, false
	}
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(password)), []byte(user.Password)) != 1 {
		return Principal{}, nil, false
	}

	payload := sessionCookiePayload{
		Username:  strings.TrimSpace(username),
		ExpiresAt: time.Now().Add(m.ttl).Unix(),
	}
	cookieValue, ok := m.encodeCookie(payload)
	if !ok {
		return Principal{}, nil, false
	}

	return sessionPrincipalFromUser(user), &http.Cookie{
		Name:     m.cookieName,
		Value:    cookieValue,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   shouldSecureCookie(request),
		MaxAge:   int(m.ttl.Seconds()),
		Expires:  time.Unix(payload.ExpiresAt, 0),
	}, true
}

func (m *SessionManager) LogoutCookie(request *http.Request) *http.Cookie {
	return &http.Cookie{
		Name:     m.cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   shouldSecureCookie(request),
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	}
}

func sessionPrincipalFromUser(user SessionUserConfig) Principal {
	actor := strings.TrimSpace(user.Actor)
	if actor == "" {
		actor = strings.TrimSpace(user.Role)
	}
	return Principal{
		Role:            strings.TrimSpace(user.Role),
		Actor:           actor,
		AuthMechanism:   "session_cookie",
		AllowedPackages: append([]string(nil), user.AllowedPackages...),
	}
}

func (m *SessionManager) encodeCookie(payload sessionCookiePayload) (string, bool) {
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return "", false
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(rawPayload)
	signature := m.sign(encodedPayload)
	return encodedPayload + "." + signature, true
}

func (m *SessionManager) decodeCookie(value string) (sessionCookiePayload, bool) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return sessionCookiePayload{}, false
	}

	encodedPayload := strings.TrimSpace(parts[0])
	signature := strings.TrimSpace(parts[1])
	expectedSignature := m.sign(encodedPayload)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSignature)) != 1 {
		return sessionCookiePayload{}, false
	}

	rawPayload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return sessionCookiePayload{}, false
	}

	var payload sessionCookiePayload
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return sessionCookiePayload{}, false
	}
	if strings.TrimSpace(payload.Username) == "" || payload.ExpiresAt == 0 {
		return sessionCookiePayload{}, false
	}
	return payload, true
}

func (m *SessionManager) sign(encodedPayload string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(encodedPayload))
	return hex.EncodeToString(mac.Sum(nil))
}

func shouldSecureCookie(request *http.Request) bool {
	if request.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(request.Header.Get("X-Forwarded-Proto")), "https")
}
