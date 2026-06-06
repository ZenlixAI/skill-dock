package catalogapi

import (
	"context"
	"net/http"
	"net/url"

	"skill_registry/app/auth"
	"skill_registry/business/catalog"
	"skill_registry/foundation/persist"
)

type auditQueryModule struct {
	audit      persist.AuditStore
	authorizer *auth.Authorizer
}

type auditQueryWorkflow interface {
	listEvents(ctx context.Context, principal auth.Principal, values url.Values) (persist.AuditPage, *catalog.Error)
}

func (m *auditQueryModule) listEvents(ctx context.Context, principal auth.Principal, values url.Values) (persist.AuditPage, *catalog.Error) {
	if m.audit == nil {
		return persist.AuditPage{}, catalog.NewError(http.StatusServiceUnavailable, "internal_error", "Audit store is not configured.", nil)
	}

	pageSize, err := parsePageSize(values)
	if err != nil {
		return persist.AuditPage{}, err
	}
	cursor, err := parseCursor(values)
	if err != nil {
		return persist.AuditPage{}, err
	}

	packageName := values.Get("package_name")
	if m.authorizer != nil {
		if authErr := m.authorizer.RequirePackageFilter(principal, packageName); authErr != nil {
			return persist.AuditPage{}, authErr.CatalogError()
		}
	}

	authDecision, authDecisionErr := parseAuthDecisionFilter(values.Get("auth_decision"))
	if authDecisionErr != nil {
		return persist.AuditPage{}, authDecisionErr
	}
	version, versionErr := parseExactVersionQueryParam(values.Get("version"), "version")
	if versionErr != nil {
		return persist.AuditPage{}, versionErr
	}

	page, listErr := m.audit.List(ctx, persist.AuditQuery{
		Action:       values.Get("action"),
		Actor:        values.Get("actor"),
		AuthDecision: authDecision,
		PackageName:  packageName,
		Version:      version,
		Outcome:      values.Get("outcome"),
		PageSize:     pageSize,
		Cursor:       cursor,
	})
	if listErr != nil {
		return persist.AuditPage{}, catalogErrorFromError(listErr)
	}
	return page, nil
}
