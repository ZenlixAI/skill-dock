package http

import (
	"context"
	stdhttp "net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/common"
	"skill_registry/app/catalogapi/response"
	"skill_registry/business/catalog"
	"skill_registry/foundation/persist"
)

type ServeCachedReadJSONFunc func(stdhttp.ResponseWriter, *stdhttp.Request, time.Duration, func() (any, *catalog.Error))
type AuthorizePackageFunc func(stdhttp.ResponseWriter, *stdhttp.Request, string) bool
type ListAuditEventsFunc func(context.Context, auth.Principal, url.Values) (persist.AuditPage, *catalog.Error)
type ListSkillsFunc func(context.Context, auth.Principal, SkillsQuery) (catalog.SkillsPage, *catalog.Error)
type ListCuratedSkillsFunc func(context.Context, auth.Principal, int, int) (catalog.CuratedSkillsPage, *catalog.Error)
type GetSkillFunc func(context.Context, string) (catalog.SkillDetail, error)
type ListVersionsFunc func(context.Context, catalog.VersionQuery) (catalog.VersionsPage, error)
type GetVersionFunc func(context.Context, string, string) (catalog.SkillVersionDetail, error)
type ResolveInstallTargetFunc func(context.Context, string, string) (catalog.InstallResolution, error)
type PublicInstallTargetFunc func(catalog.InstallResolution) catalog.InstallResolution

type Handlers struct {
	AuthorizePackage     AuthorizePackageFunc
	ServeCachedReadJSON  ServeCachedReadJSONFunc
	ReadListCacheTTL     time.Duration
	ReadDetailCacheTTL   time.Duration
	ListAuditEvents      ListAuditEventsFunc
	ListSkillsFunc       ListSkillsFunc
	ListCuratedSkills    ListCuratedSkillsFunc
	GetSkillFunc         GetSkillFunc
	ListVersionsFunc     ListVersionsFunc
	GetVersionFunc       GetVersionFunc
	ResolveInstallTarget ResolveInstallTargetFunc
	PublicInstallTarget  PublicInstallTargetFunc
}

func New(config Handlers) Handlers {
	return config
}

func (h Handlers) Skills(context *gin.Context) {
	query, err := BindSkillsQuery(context)
	if err != nil {
		response.WriteCatalogError(context, err)
		return
	}
	if query.PackageName != "" && !h.authorizePackage(context, query.PackageName) {
		return
	}

	h.serveCachedReadJSON(context, h.ReadListCacheTTL, func() (any, *catalog.Error) {
		if h.ListSkillsFunc == nil {
			return catalog.SkillsPage{}, nil
		}
		return h.ListSkillsFunc(context.Request.Context(), auth.PrincipalFromContext(context.Request.Context()), query)
	})
}

func (h Handlers) AuditEvents(context *gin.Context) {
	packageName := context.Query("package_name")
	if packageName != "" && !h.authorizePackage(context, packageName) {
		return
	}

	h.serveCachedReadJSON(context, h.ReadListCacheTTL, func() (any, *catalog.Error) {
		if h.ListAuditEvents == nil {
			return persist.AuditPage{}, nil
		}
		return h.ListAuditEvents(context.Request.Context(), auth.PrincipalFromContext(context.Request.Context()), context.Request.URL.Query())
	})
}

func (h Handlers) SkillsSearch(context *gin.Context) {
	h.Skills(context)
}

func (h Handlers) CuratedSkills(context *gin.Context) {
	query, err := BindCuratedSkillsQuery(context)
	if err != nil {
		response.WriteCatalogError(context, err)
		return
	}
	pageSize, pageSizeErr := common.ParsePageSizeValue(query.PageSize)
	if pageSizeErr != nil {
		response.WriteCatalogError(context, pageSizeErr)
		return
	}
	cursor, cursorErr := common.ParseCursorValue(query.Cursor)
	if cursorErr != nil {
		response.WriteCatalogError(context, cursorErr)
		return
	}

	h.serveCachedReadJSON(context, h.ReadListCacheTTL, func() (any, *catalog.Error) {
		if h.ListCuratedSkills == nil {
			return catalog.CuratedSkillsPage{}, nil
		}
		return h.ListCuratedSkills(context.Request.Context(), auth.PrincipalFromContext(context.Request.Context()), pageSize, cursor)
	})
}

func (h Handlers) SkillDetail(context *gin.Context) {
	uri, err := BindPackageURI(context)
	if err != nil {
		response.WriteCatalogError(context, err)
		return
	}
	packageName, packageErr := DecodePackageName(uri.Package)
	if packageErr != nil {
		response.WriteCatalogError(context, packageErr)
		return
	}
	if !h.authorizePackage(context, packageName) {
		return
	}

	h.serveCachedReadJSON(context, h.ReadDetailCacheTTL, func() (any, *catalog.Error) {
		if h.GetSkillFunc == nil {
			return catalog.SkillDetail{}, nil
		}
		payload, getErr := h.GetSkillFunc(context.Request.Context(), packageName)
		if getErr != nil {
			return nil, common.CatalogErrorFromError(getErr)
		}
		return payload, nil
	})
}

func (h Handlers) VersionList(context *gin.Context) {
	uri, err := BindPackageURI(context)
	if err != nil {
		response.WriteCatalogError(context, err)
		return
	}
	query, queryErr := BindVersionListQuery(context)
	if queryErr != nil {
		response.WriteCatalogError(context, queryErr)
		return
	}
	packageName, packageErr := DecodePackageName(uri.Package)
	if packageErr != nil {
		response.WriteCatalogError(context, packageErr)
		return
	}
	if !h.authorizePackage(context, packageName) {
		return
	}

	includeDeprecated, includeDeprecatedErr := common.ParseBooleanQueryParam(query.IncludeDeprecated, "include_deprecated", true)
	if includeDeprecatedErr != nil {
		response.WriteCatalogError(context, includeDeprecatedErr)
		return
	}
	includeYanked, includeYankedErr := common.ParseBooleanQueryParam(query.IncludeYanked, "include_yanked", false)
	if includeYankedErr != nil {
		response.WriteCatalogError(context, includeYankedErr)
		return
	}
	pageSize, pageSizeErr := common.ParsePageSizeValue(query.PageSize)
	if pageSizeErr != nil {
		response.WriteCatalogError(context, pageSizeErr)
		return
	}
	cursor, cursorErr := common.ParseCursorValue(query.Cursor)
	if cursorErr != nil {
		response.WriteCatalogError(context, cursorErr)
		return
	}

	h.serveCachedReadJSON(context, h.ReadDetailCacheTTL, func() (any, *catalog.Error) {
		if h.ListVersionsFunc == nil {
			return catalog.VersionsPage{}, nil
		}
		payload, listErr := h.ListVersionsFunc(context.Request.Context(), catalog.VersionQuery{
			PackageName:       packageName,
			IncludeDeprecated: includeDeprecated,
			IncludeYanked:     includeYanked,
			PageSize:          pageSize,
			Cursor:            cursor,
		})
		if listErr != nil {
			return nil, common.CatalogErrorFromError(listErr)
		}
		return payload, nil
	})
}

func (h Handlers) VersionDetail(context *gin.Context) {
	uri, err := BindVersionURI(context)
	if err != nil {
		response.WriteCatalogError(context, err)
		return
	}
	packageName, packageErr := DecodePackageName(uri.Package)
	if packageErr != nil {
		response.WriteCatalogError(context, packageErr)
		return
	}
	if !h.authorizePackage(context, packageName) {
		return
	}
	version, versionErr := common.DecodeVersion(uri.Version)
	if versionErr != nil {
		response.WriteCatalogError(context, versionErr)
		return
	}

	h.serveCachedReadJSON(context, h.ReadDetailCacheTTL, func() (any, *catalog.Error) {
		if h.GetVersionFunc == nil {
			return catalog.SkillVersionDetail{}, nil
		}
		payload, getErr := h.GetVersionFunc(context.Request.Context(), packageName, version)
		if getErr != nil {
			return nil, common.CatalogErrorFromError(getErr)
		}
		return payload, nil
	})
}

func (h Handlers) InstallTarget(context *gin.Context) {
	uri, err := BindPackageURI(context)
	if err != nil {
		response.WriteCatalogError(context, err)
		return
	}
	query, queryErr := BindInstallTargetQuery(context)
	if queryErr != nil {
		response.WriteCatalogError(context, queryErr)
		return
	}
	packageName, packageErr := DecodePackageName(uri.Package)
	if packageErr != nil {
		response.WriteCatalogError(context, packageErr)
		return
	}
	if !h.authorizePackage(context, packageName) {
		return
	}

	h.serveCachedReadJSON(context, h.ReadDetailCacheTTL, func() (any, *catalog.Error) {
		if h.ResolveInstallTarget == nil {
			return catalog.InstallResolution{}, nil
		}
		payload, resolveErr := h.ResolveInstallTarget(context.Request.Context(), packageName, query.Version)
		if resolveErr != nil {
			return nil, common.CatalogErrorFromError(resolveErr)
		}
		if h.PublicInstallTarget != nil {
			return h.PublicInstallTarget(payload), nil
		}
		return payload, nil
	})
}

func (h Handlers) authorizePackage(context *gin.Context, packageName string) bool {
	if h.AuthorizePackage == nil {
		return true
	}
	return h.AuthorizePackage(context.Writer, context.Request, packageName)
}

func (h Handlers) serveCachedReadJSON(context *gin.Context, ttl time.Duration, load func() (any, *catalog.Error)) {
	if h.ServeCachedReadJSON != nil {
		h.ServeCachedReadJSON(context.Writer, context.Request, ttl, load)
		return
	}
	payload, err := load()
	if err != nil {
		response.WriteCatalogError(context, err)
		return
	}
	response.WriteJSON(context, stdhttp.StatusOK, payload)
}
