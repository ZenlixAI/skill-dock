package binding

import (
	"github.com/gin-gonic/gin"

	"skill_registry/business/catalog"
	internalcataloghttp "skill_registry/internal/catalog/http"
)

type SkillsQuery = internalcataloghttp.SkillsQuery
type CuratedSkillsQuery = internalcataloghttp.CuratedSkillsQuery
type GovernanceChangesQuery = internalcataloghttp.GovernanceChangesQuery
type VersionListQuery = internalcataloghttp.VersionListQuery
type InstallTargetQuery = internalcataloghttp.InstallTargetQuery
type PackageURI = internalcataloghttp.PackageURI
type VersionURI = internalcataloghttp.VersionURI

func BindSkillsQuery(context *gin.Context) (SkillsQuery, *catalog.Error) {
	return internalcataloghttp.BindSkillsQuery(context)
}

func BindCuratedSkillsQuery(context *gin.Context) (CuratedSkillsQuery, *catalog.Error) {
	return internalcataloghttp.BindCuratedSkillsQuery(context)
}

func BindGovernanceChangesQuery(context *gin.Context) (GovernanceChangesQuery, *catalog.Error) {
	return internalcataloghttp.BindGovernanceChangesQuery(context)
}

func BindVersionListQuery(context *gin.Context) (VersionListQuery, *catalog.Error) {
	return internalcataloghttp.BindVersionListQuery(context)
}

func BindInstallTargetQuery(context *gin.Context) (InstallTargetQuery, *catalog.Error) {
	return internalcataloghttp.BindInstallTargetQuery(context)
}

func BindPackageURI(context *gin.Context) (PackageURI, *catalog.Error) {
	return internalcataloghttp.BindPackageURI(context)
}

func BindVersionURI(context *gin.Context) (VersionURI, *catalog.Error) {
	return internalcataloghttp.BindVersionURI(context)
}

func DecodePackageName(raw string) (string, *catalog.Error) {
	return internalcataloghttp.DecodePackageName(raw)
}
