package access

import (
	"context"
	"strings"
	"time"

	"skill_registry/internal/platform/auth"
)

type Action string

const (
	ActionRead     Action = "read"
	ActionInstall  Action = "install"
	ActionPublish  Action = "publish"
	ActionManage   Action = "manage"
	ActionGovern   Action = "govern"
	ActionGrant    Action = "grant"
	ActionTransfer Action = "transfer"
)

type Decision struct {
	Allowed bool
	Reason  string
}

type Decider struct {
	store Store
}

func NewDecider(store Store) *Decider {
	return &Decider{store: store}
}

func (d *Decider) Decide(ctx context.Context, principal auth.Principal, action Action, resource SkillAccess) Decision {
	visibility := strings.TrimSpace(resource.Visibility)
	if visibility == "" {
		visibility = "public"
	}
	if (action == ActionRead || action == ActionInstall) && visibility == "public" {
		return Decision{Allowed: true, Reason: "public"}
	}
	if principal.IsAnonymous() {
		return Decision{Allowed: false, Reason: "anonymous"}
	}
	if principal.PlatformAdmin {
		return Decision{Allowed: true, Reason: "platform_admin"}
	}
	if principal.IsService() && (principal.Role == "reader" || principal.Role == "publisher" || principal.Role == "operator") {
		return Decision{Allowed: true, Reason: "legacy_service"}
	}

	if resource.OwnerType == "user" && resource.OwnerID == principal.UserID {
		return Decision{Allowed: true, Reason: "owner"}
	}

	tenantMatches := resource.OrganizationID != "" && principal.TenantID == resource.OrganizationID
	if !tenantMatches && visibility != "public" {
		return Decision{Allowed: false, Reason: "tenant_mismatch"}
	}

	if tenantMatches {
		switch action {
		case ActionRead, ActionInstall:
			if visibility == "organization" {
				return Decision{Allowed: true, Reason: "tenant_member"}
			}
			if visibility == "private" && (principal.HasScope("skill:manage") || principal.HasScope("org:manage") || principal.HasScope("skill:govern")) {
				return Decision{Allowed: true, Reason: "tenant_manager"}
			}
			if visibility == "private" && d != nil && d.store != nil {
				if maintainer, err := d.store.IsMaintainer(ctx, resource.PackageName, principal.UserID); err == nil && maintainer {
					return Decision{Allowed: true, Reason: "maintainer"}
				}
			}
		case ActionPublish:
			if principal.HasScope("skill:publish") {
				return Decision{Allowed: true, Reason: "tenant_publisher"}
			}
		case ActionManage:
			if principal.HasScope("skill:manage") || principal.HasScope("org:manage") {
				return Decision{Allowed: true, Reason: "tenant_manager"}
			}
		case ActionGovern:
			if principal.HasScope("skill:govern") || principal.HasScope("org:manage") {
				return Decision{Allowed: true, Reason: "tenant_governor"}
			}
		case ActionGrant, ActionTransfer:
			if principal.HasScope("org:manage") {
				return Decision{Allowed: true, Reason: "tenant_owner"}
			}
		}
		if d != nil && d.store != nil && (action == ActionPublish || action == ActionManage) {
			if maintainer, err := d.store.IsMaintainer(ctx, resource.PackageName, principal.UserID); err == nil && maintainer {
				return Decision{Allowed: true, Reason: "maintainer"}
			}
		}
	}

	if d != nil && d.store != nil && resource.OrganizationID != "" && principal.UserID != "" {
		grants, err := d.store.ListGrants(ctx, resource.OrganizationID, resource.PackageName)
		if err == nil && grantAllows(grants, principal.UserID, action) {
			return Decision{Allowed: true, Reason: "grant"}
		}
	}

	return Decision{Allowed: false, Reason: "forbidden"}
}

func grantAllows(grants []Grant, userID string, action Action) bool {
	required := permissionRank(action)
	if required == 0 {
		return false
	}
	now := time.Now().UTC()
	for _, grant := range grants {
		if grant.SubjectType != "user" || grant.SubjectID != userID {
			continue
		}
		if grant.ExpiresAt != nil && !grant.ExpiresAt.After(now) {
			continue
		}
		if permissionRank(Action(grant.Permission)) >= required {
			return true
		}
	}
	return false
}

func permissionRank(action Action) int {
	switch action {
	case ActionRead:
		return 1
	case ActionInstall:
		return 2
	case ActionPublish:
		return 3
	case ActionManage:
		return 4
	case ActionGovern:
		return 5
	case ActionGrant, ActionTransfer:
		return 6
	default:
		return 0
	}
}
