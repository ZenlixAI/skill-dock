package cataloghandlers

import internalcataloghttp "skill_registry/internal/catalog/http"

type ServeCachedReadJSONFunc = internalcataloghttp.ServeCachedReadJSONFunc
type AuthorizePackageFunc = internalcataloghttp.AuthorizePackageFunc
type ListAuditEventsFunc = internalcataloghttp.ListAuditEventsFunc
type ListSkillsFunc = internalcataloghttp.ListSkillsFunc
type ListCuratedSkillsFunc = internalcataloghttp.ListCuratedSkillsFunc
type ListGovernanceChangesFunc = internalcataloghttp.ListGovernanceChangesFunc
type GetSkillFunc = internalcataloghttp.GetSkillFunc
type ListVersionsFunc = internalcataloghttp.ListVersionsFunc
type GetVersionFunc = internalcataloghttp.GetVersionFunc
type ResolveInstallTargetFunc = internalcataloghttp.ResolveInstallTargetFunc
type PublicInstallTargetFunc = internalcataloghttp.PublicInstallTargetFunc
type Handlers = internalcataloghttp.Handlers

func New(config Handlers) Handlers {
	return internalcataloghttp.New(config)
}
