package catalogapi

import (
	"context"

	"skill_registry/app/auth"
	"skill_registry/business/catalog"
)

type governanceModule struct {
	repo       catalog.Repository
	authorizer *auth.Authorizer
}

func (m *governanceModule) listChanges(ctx context.Context, principal auth.Principal, query catalog.GovernanceQuery) (catalog.GovernancePage, *catalog.Error) {
	if authErr := m.authorizer.RequirePackageFilter(principal, query.PackageName); authErr != nil {
		return catalog.GovernancePage{}, authErr.CatalogError()
	}

	response, err := m.repo.ListGovernanceChanges(ctx, query)
	if err != nil {
		return catalog.GovernancePage{}, catalogErrorFromError(err)
	}
	response.Items = auth.FilterGovernanceRecordsByPrincipal(response.Items, principal)
	return response, nil
}

func catalogErrorFromError(err error) *catalog.Error {
	if err == nil {
		return nil
	}
	if catalogErr, ok := err.(*catalog.Error); ok {
		return catalogErr
	}
	return catalog.NewError(500, "internal_error", "Internal server error.", nil)
}
