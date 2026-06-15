package auth

import platformauth "skill_registry/internal/platform/auth"

const SessionCookieName = platformauth.SessionCookieName

type SessionUserConfig = platformauth.SessionUserConfig
type SessionManager = platformauth.SessionManager

var NewSessionManager = platformauth.NewSessionManager
