package catalogapi

import (
	"context"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/common"
	"skill_registry/business/catalog"
)

type catalogGovernanceService interface {
	ListGovernanceChanges(ctx context.Context, query catalog.GovernanceQuery) (catalog.GovernancePage, error)
}

type governanceModule struct {
	service    catalogGovernanceService
	authorizer *auth.Authorizer
}

func (m *governanceModule) listChanges(ctx context.Context, principal auth.Principal, query catalog.GovernanceQuery) (catalog.GovernancePage, *catalog.Error) {
	if authErr := m.authorizer.RequirePackageFilter(principal, query.PackageName); authErr != nil {
		return catalog.GovernancePage{}, authErr.CatalogError()
	}

	response, err := m.service.ListGovernanceChanges(ctx, query)
	if err != nil {
		return catalog.GovernancePage{}, common.CatalogErrorFromError(err)
	}
	response.Items = auth.FilterGovernanceRecordsByPrincipal(response.Items, principal)
	return response, nil
}
