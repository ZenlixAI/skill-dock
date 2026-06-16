package auth

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	catalog "skill_registry/internal/catalog/domain"
)

type Config struct {
	ReaderToken    string
	AuthoringToken string
	PublisherToken string
	Tokens         map[string]TokenConfig
}

type TokenConfig struct {
	Role            string   `json:"role"`
	Actor           string   `json:"actor,omitempty"`
	AllowedPackages []string `json:"allowedPackages,omitempty"`
}

type Principal struct {
	Role            string
	Actor           string
	AuthMechanism   string
	AllowedPackages []string
}

type Action string

const (
	ActionReadCatalog            Action = "catalog.read"
	ActionReadAuditEvents        Action = "audit.read"
	ActionReadSession            Action = "session.read"
	ActionReadHealthz            Action = "system.healthz.read"
	ActionReadReadyz             Action = "system.readyz.read"
	ActionReadMetrics            Action = "system.metrics.read"
	ActionPackagingPrepare       Action = "packaging.prepare"
	ActionPackagingValidate      Action = "packaging.validate"
	ActionPackagingPack          Action = "packaging.pack"
	ActionPackagingPublish       Action = "packaging.publish"
	ActionCreateReleaseRecord    Action = "catalog.release_record.create"
	ActionApplySkillGovernance   Action = "catalog.governance.skill.apply"
	ActionApplyVersionGovernance Action = "catalog.governance.version.apply"
)

type AuthorizationError struct {
	Status  int
	Code    string
	Message string
	Details map[string]any
}

type Authorizer struct {
	tokens          map[string]Principal
	allowHeaderAuth bool
	capabilities    map[Action]map[string]struct{}
}

func NewAuthorizer(config Config) *Authorizer {
	tokens := make(map[string]Principal)
	for token, tokenConfig := range config.Tokens {
		role := strings.TrimSpace(tokenConfig.Role)
		if strings.TrimSpace(token) == "" || role == "" {
			continue
		}
		actor := strings.TrimSpace(tokenConfig.Actor)
		if actor == "" {
			actor = role
		}
		tokens[strings.TrimSpace(token)] = Principal{
			Role:            role,
			Actor:           actor,
			AuthMechanism:   "configured_token",
			AllowedPackages: append([]string(nil), tokenConfig.AllowedPackages...),
		}
	}
	if len(tokens) == 0 {
		if strings.TrimSpace(config.ReaderToken) != "" {
			tokens[strings.TrimSpace(config.ReaderToken)] = Principal{Role: "reader", Actor: "reader", AuthMechanism: "legacy_token"}
		}
		if strings.TrimSpace(config.AuthoringToken) != "" {
			tokens[strings.TrimSpace(config.AuthoringToken)] = Principal{Role: "authoring", Actor: "authoring", AuthMechanism: "legacy_token"}
		}
		if strings.TrimSpace(config.PublisherToken) != "" {
			tokens[strings.TrimSpace(config.PublisherToken)] = Principal{Role: "publisher", Actor: "publisher", AuthMechanism: "legacy_token"}
		}
	}

	return &Authorizer{
		tokens:          tokens,
		allowHeaderAuth: len(tokens) == 0,
		capabilities:    defaultCapabilities(),
	}
}

func (a *Authorizer) Authenticate(request *http.Request) (Principal, bool) {
	authorization := strings.TrimSpace(request.Header.Get("Authorization"))
	if authorization != "" {
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authorization, bearerPrefix) {
			return Principal{}, false
		}
		token := strings.TrimSpace(strings.TrimPrefix(authorization, bearerPrefix))
		if token == "" {
			return Principal{}, false
		}
		current, ok := a.tokens[token]
		return current, ok
	}
	if !a.allowHeaderAuth {
		return Principal{}, false
	}
	role := strings.TrimSpace(request.Header.Get("X-Role"))
	if role == "" {
		return Principal{}, false
	}
	actor := strings.TrimSpace(request.Header.Get("X-Actor"))
	if actor == "" {
		actor = role
	}
	return Principal{
		Role:          role,
		Actor:         actor,
		AuthMechanism: "dev_header",
	}, true
}

func (a *Authorizer) AuthorizeRole(current Principal, allowedRoles ...string) *AuthorizationError {
	for _, allowedRole := range allowedRoles {
		if current.Role == allowedRole {
			return nil
		}
	}
	return &AuthorizationError{
		Status:  http.StatusForbidden,
		Code:    "forbidden",
		Message: "Role is not allowed for this endpoint.",
		Details: map[string]any{"role": current.Role},
	}
}

func (a *Authorizer) AuthorizeAction(current Principal, action Action) *AuthorizationError {
	allowedRoles, ok := a.capabilities[action]
	if !ok {
		return &AuthorizationError{
			Status:  http.StatusForbidden,
			Code:    "forbidden",
			Message: "Action is not allowed for this endpoint.",
			Details: map[string]any{"action": string(action), "role": current.Role},
		}
	}
	if _, ok := allowedRoles[current.Role]; ok {
		return nil
	}
	return &AuthorizationError{
		Status:  http.StatusForbidden,
		Code:    "forbidden",
		Message: "Action is not allowed for this endpoint.",
		Details: map[string]any{"action": string(action), "role": current.Role},
	}
}

func (a *Authorizer) AuthorizePackage(current Principal, packageName string) *AuthorizationError {
	if current.CanAccessPackage(packageName) {
		return nil
	}
	return &AuthorizationError{
		Status:  http.StatusForbidden,
		Code:    "package_forbidden",
		Message: "Actor is not allowed to access this package.",
		Details: map[string]any{"packageName": packageName},
	}
}

func (a *Authorizer) RequirePackageFilter(current Principal, packageName string) *AuthorizationError {
	if len(current.AllowedPackages) == 0 || strings.TrimSpace(packageName) != "" {
		return nil
	}
	return &AuthorizationError{
		Status:  http.StatusForbidden,
		Code:    "package_forbidden",
		Message: "package_name filter is required for package-scoped principals.",
	}
}

func (e *AuthorizationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AuthorizationError) CatalogError() *catalog.Error {
	return catalog.NewError(e.Status, e.Code, e.Message, e.Details)
}

func (p Principal) CanAccessPackage(packageName string) bool {
	if strings.TrimSpace(packageName) == "" || len(p.AllowedPackages) == 0 {
		return true
	}
	for _, pattern := range p.AllowedPackages {
		if matchesPackagePattern(pattern, packageName) {
			return true
		}
	}
	return false
}

func (a *Authorizer) AllowedActionsForRole(role string) []string {
	allowed := make([]string, 0)
	for action, roles := range a.capabilities {
		if _, ok := roles[role]; ok {
			allowed = append(allowed, string(action))
		}
	}
	sort.Strings(allowed)
	return allowed
}

func FilterSkillSummariesByPrincipal(items []catalog.SkillSummary, current Principal) []catalog.SkillSummary {
	if len(current.AllowedPackages) == 0 {
		return items
	}
	filtered := make([]catalog.SkillSummary, 0, len(items))
	for _, item := range items {
		if current.CanAccessPackage(item.PackageName) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func FilterCuratedSkillSummariesByPrincipal(items []catalog.CuratedSkillSummary, current Principal) []catalog.CuratedSkillSummary {
	if len(current.AllowedPackages) == 0 {
		return items
	}
	filtered := make([]catalog.CuratedSkillSummary, 0, len(items))
	for _, item := range items {
		if current.CanAccessPackage(item.PackageName) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func FilterGovernanceRecordsByPrincipal(items []catalog.GovernanceRecord, current Principal) []catalog.GovernanceRecord {
	if len(current.AllowedPackages) == 0 {
		return items
	}
	filtered := make([]catalog.GovernanceRecord, 0, len(items))
	for _, item := range items {
		if current.CanAccessPackage(item.PackageName) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func matchesPackagePattern(pattern string, packageName string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(packageName, strings.TrimSuffix(pattern, "*"))
	}
	return packageName == pattern
}

func defaultCapabilities() map[Action]map[string]struct{} {
	return map[Action]map[string]struct{}{
		ActionReadCatalog: {
			"reader":    {},
			"publisher": {},
			"operator":  {},
		},
		ActionReadAuditEvents: {
			"reader":    {},
			"publisher": {},
			"operator":  {},
		},
		ActionReadSession: {
			"reader":    {},
			"authoring": {},
			"publisher": {},
			"operator":  {},
		},
		ActionPackagingPrepare: {
			"authoring": {},
			"publisher": {},
			"operator":  {},
		},
		ActionPackagingValidate: {
			"authoring": {},
			"publisher": {},
			"operator":  {},
		},
		ActionPackagingPack: {
			"authoring": {},
			"publisher": {},
			"operator":  {},
		},
		ActionPackagingPublish: {
			"publisher": {},
			"operator":  {},
		},
		ActionCreateReleaseRecord: {
			"publisher": {},
			"operator":  {},
		},
		ActionApplySkillGovernance: {
			"publisher": {},
			"operator":  {},
		},
		ActionApplyVersionGovernance: {
			"publisher": {},
			"operator":  {},
		},
	}
}
