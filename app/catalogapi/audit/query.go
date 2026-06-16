package audit

import (
	"context"
	"net/http"
	"net/url"

	catalog "skill_registry/internal/catalog/domain"
	internalcataloghttp "skill_registry/internal/catalog/http"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
	persist "skill_registry/internal/platform/persistence"
)

type QueryModule struct {
	audit      persist.AuditStore
	authorizer *auth.Authorizer
}

func NewQueryModule(store persist.AuditStore, authorizer *auth.Authorizer) *QueryModule {
	return &QueryModule{audit: store, authorizer: authorizer}
}

func (m *QueryModule) ListEvents(ctx context.Context, principal auth.Principal, values url.Values) (persist.AuditPage, *catalog.Error) {
	if m.audit == nil {
		return persist.AuditPage{}, catalog.NewError(http.StatusServiceUnavailable, "internal_error", "Audit store is not configured.", nil)
	}

	pageSize, err := internalcataloghttp.ParsePageSize(values)
	if err != nil {
		return persist.AuditPage{}, err
	}
	cursor, err := internalcataloghttp.ParseCursor(values)
	if err != nil {
		return persist.AuditPage{}, err
	}

	packageName := values.Get("package_name")
	if m.authorizer != nil {
		if authErr := m.authorizer.RequirePackageFilter(principal, packageName); authErr != nil {
			return persist.AuditPage{}, authErr.CatalogError()
		}
	}

	authDecision, authDecisionErr := internalcataloghttp.ParseAuthDecisionFilter(values.Get("auth_decision"))
	if authDecisionErr != nil {
		return persist.AuditPage{}, authDecisionErr
	}
	version, versionErr := internalcataloghttp.ParseExactVersionQueryParam(values.Get("version"), "version")
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
		return persist.AuditPage{}, platformhttp.CatalogErrorFromError(listErr)
	}
	return page, nil
}
