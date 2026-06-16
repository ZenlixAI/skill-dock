package http

import (
	"net/url"

	"github.com/gin-gonic/gin"

	catalog "skill_registry/internal/catalog/domain"
)

type SkillsQuery struct {
	Query          string `form:"q"`
	PackageName    string `form:"package_name"`
	SourceType     string `form:"source_type"`
	IncludeRetired string `form:"include_retired"`
	Sort           string `form:"sort"`
	PageSize       string `form:"pageSize"`
	Cursor         string `form:"cursor"`
}

type CuratedSkillsQuery struct {
	PageSize string `form:"pageSize"`
	Cursor   string `form:"cursor"`
}

type GovernanceChangesQuery struct {
	PackageName string `form:"package_name"`
	TargetType  string `form:"target_type"`
	Version     string `form:"version"`
	PageSize    string `form:"pageSize"`
	Cursor      string `form:"cursor"`
}

type VersionListQuery struct {
	IncludeDeprecated string `form:"include_deprecated"`
	IncludeYanked     string `form:"include_yanked"`
	PageSize          string `form:"pageSize"`
	Cursor            string `form:"cursor"`
}

type InstallTargetQuery struct {
	Version string `form:"version"`
}

type PackageURI struct {
	Package string `uri:"package"`
}

type VersionURI struct {
	Package string `uri:"package"`
	Version string `uri:"version"`
}

func BindSkillsQuery(context *gin.Context) (SkillsQuery, *catalog.Error) {
	return bindQuery[SkillsQuery](context)
}

func BindCuratedSkillsQuery(context *gin.Context) (CuratedSkillsQuery, *catalog.Error) {
	return bindQuery[CuratedSkillsQuery](context)
}

func BindGovernanceChangesQuery(context *gin.Context) (GovernanceChangesQuery, *catalog.Error) {
	return bindQuery[GovernanceChangesQuery](context)
}

func BindVersionListQuery(context *gin.Context) (VersionListQuery, *catalog.Error) {
	return bindQuery[VersionListQuery](context)
}

func BindInstallTargetQuery(context *gin.Context) (InstallTargetQuery, *catalog.Error) {
	return bindQuery[InstallTargetQuery](context)
}

func BindPackageURI(context *gin.Context) (PackageURI, *catalog.Error) {
	return bindURI[PackageURI](context)
}

func BindVersionURI(context *gin.Context) (VersionURI, *catalog.Error) {
	return bindURI[VersionURI](context)
}

func DecodePackageName(raw string) (string, *catalog.Error) {
	packageName, err := url.PathUnescape(raw)
	if err != nil {
		return "", catalog.NewError(400, "invalid_request", "package path is not valid URL encoding.", map[string]any{"package": raw})
	}
	return packageName, nil
}

func bindQuery[T any](context *gin.Context) (T, *catalog.Error) {
	var query T
	if err := context.ShouldBindQuery(&query); err != nil {
		return query, catalog.NewError(400, "invalid_request", "query parameters are invalid.", map[string]any{"error": err.Error()})
	}
	return query, nil
}

func bindURI[T any](context *gin.Context) (T, *catalog.Error) {
	var uri T
	if err := context.ShouldBindUri(&uri); err != nil {
		return uri, catalog.NewError(400, "invalid_request", "path parameters are invalid.", map[string]any{"error": err.Error()})
	}
	return uri, nil
}
