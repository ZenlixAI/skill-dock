package http

import (
	"context"
	"io"
	stdhttp "net/http"
	"time"

	"github.com/gin-gonic/gin"

	"skill_registry/app/catalogapi/common"
	"skill_registry/app/catalogapi/response"
	"skill_registry/business/catalog"
	cataloghttp "skill_registry/internal/catalog/http"
	"skill_registry/internal/platform/auth"
)

type ServeCachedReadJSONFunc func(stdhttp.ResponseWriter, *stdhttp.Request, time.Duration, func() (any, *catalog.Error))
type AuthorizePackageFunc func(stdhttp.ResponseWriter, *stdhttp.Request, string) bool
type AuthorizeActionFunc func(stdhttp.ResponseWriter, *stdhttp.Request, auth.Action, string, string) bool
type AcquireWriteLimiterFunc func() (func(), *catalog.Error)
type ListGovernanceChangesFunc func(context.Context, auth.Principal, catalog.GovernanceQuery) (catalog.GovernancePage, *catalog.Error)
type CreateReleaseRecordFunc func(ctx any, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error)
type ApplySkillGovernanceChangeFunc func(ctx any, principal auth.Principal, packageName string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error)
type ApplyVersionGovernanceChangeFunc func(ctx any, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error)

type Handlers struct {
	ServeCachedReadJSON        ServeCachedReadJSONFunc
	ReadListCacheTTL           time.Duration
	AuthorizePackage           AuthorizePackageFunc
	AuthorizeAction            AuthorizeActionFunc
	AcquireWriteLimiter        AcquireWriteLimiterFunc
	ListGovernance             ListGovernanceChangesFunc
	CreateReleaseRecord        func(request *stdhttp.Request, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error)
	ApplySkillGovernanceChange func(request *stdhttp.Request, principal auth.Principal, packageName string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error)
	ApplyVersionGovernance     func(request *stdhttp.Request, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error)
}

type Config struct {
	ServesReadRoutes  bool
	ServesWriteRoutes bool
	GovernanceChanges gin.HandlerFunc
	SkillGovernance   gin.HandlerFunc
	ReleaseRecord     gin.HandlerFunc
	VersionGovernance gin.HandlerFunc
}

func New(config Handlers) Handlers {
	return config
}

func RegisterRoutes(engine *gin.Engine, config Config) {
	if config.ServesReadRoutes {
		registerHeadCompatibleRoute(engine, "/api/v1/governance-changes", config.GovernanceChanges)
	}

	if !config.ServesWriteRoutes {
		return
	}

	engine.POST("/api/v1/skills/:package/governance-changes", config.SkillGovernance)
	engine.PUT("/api/v1/skills/:package/versions/:version/release-record", config.ReleaseRecord)
	engine.POST("/api/v1/skills/:package/versions/:version/governance-changes", config.VersionGovernance)
}

func (h Handlers) GovernanceChanges(context *gin.Context) {
	query, err := cataloghttp.BindGovernanceChangesQuery(context)
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
	if query.PackageName != "" && !h.authorizePackage(context, query.PackageName) {
		return
	}
	targetType, targetTypeErr := common.ParseGovernanceTargetTypeFilter(query.TargetType)
	if targetTypeErr != nil {
		response.WriteCatalogError(context, targetTypeErr)
		return
	}
	versionFilter, versionFilterErr := common.ParseExactVersionQueryParam(query.Version, "version")
	if versionFilterErr != nil {
		response.WriteCatalogError(context, versionFilterErr)
		return
	}

	h.serveCachedReadJSON(context, h.ReadListCacheTTL, func() (any, *catalog.Error) {
		if h.ListGovernance == nil {
			return catalog.GovernancePage{}, nil
		}
		return h.ListGovernance(context.Request.Context(), auth.PrincipalFromContext(context.Request.Context()), catalog.GovernanceQuery{
			PackageName: query.PackageName,
			Version:     versionFilter,
			TargetType:  targetType,
			PageSize:    pageSize,
			Cursor:      cursor,
		})
	})
}

func (h Handlers) SkillGovernance(context *gin.Context) {
	uri, err := cataloghttp.BindPackageURI(context)
	if err != nil {
		response.WriteCatalogError(context, err)
		return
	}
	packageName, packageErr := cataloghttp.DecodePackageName(uri.Package)
	if packageErr != nil {
		response.WriteCatalogError(context, packageErr)
		return
	}
	if !h.authorizePackage(context, packageName) {
		return
	}
	if !h.authorizeAction(context, auth.ActionApplySkillGovernance, packageName, "") {
		return
	}

	rawBody, readErr := io.ReadAll(context.Request.Body)
	if readErr != nil {
		response.WriteCatalogError(context, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": readErr.Error()}))
		return
	}
	release := h.acquireWriteLimiter(context)
	if release == nil {
		return
	}
	defer release()

	if h.ApplySkillGovernanceChange == nil {
		response.WriteJSON(context, stdhttp.StatusCreated, catalog.GovernanceChangeResponse{})
		return
	}
	payload, status, handlerErr := h.ApplySkillGovernanceChange(context.Request, auth.PrincipalFromContext(context.Request.Context()), packageName, rawBody)
	if handlerErr != nil {
		response.WriteCatalogError(context, handlerErr)
		return
	}
	response.WriteJSON(context, status, payload)
}

func (h Handlers) VersionGovernance(context *gin.Context) {
	uri, err := cataloghttp.BindVersionURI(context)
	if err != nil {
		response.WriteCatalogError(context, err)
		return
	}
	packageName, packageErr := cataloghttp.DecodePackageName(uri.Package)
	if packageErr != nil {
		response.WriteCatalogError(context, packageErr)
		return
	}
	if !h.authorizePackage(context, packageName) {
		return
	}
	if !h.authorizeAction(context, auth.ActionApplyVersionGovernance, packageName, uri.Version) {
		return
	}

	rawBody, readErr := io.ReadAll(context.Request.Body)
	if readErr != nil {
		response.WriteCatalogError(context, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": readErr.Error()}))
		return
	}
	release := h.acquireWriteLimiter(context)
	if release == nil {
		return
	}
	defer release()

	if h.ApplyVersionGovernance == nil {
		response.WriteJSON(context, stdhttp.StatusCreated, catalog.GovernanceChangeResponse{})
		return
	}
	payload, status, handlerErr := h.ApplyVersionGovernance(context.Request, auth.PrincipalFromContext(context.Request.Context()), packageName, uri.Version, rawBody)
	if handlerErr != nil {
		response.WriteCatalogError(context, handlerErr)
		return
	}
	response.WriteJSON(context, status, payload)
}

func (h Handlers) ReleaseRecord(context *gin.Context) {
	uri, err := cataloghttp.BindVersionURI(context)
	if err != nil {
		response.WriteCatalogError(context, err)
		return
	}
	packageName, packageErr := cataloghttp.DecodePackageName(uri.Package)
	if packageErr != nil {
		response.WriteCatalogError(context, packageErr)
		return
	}
	if !h.authorizePackage(context, packageName) {
		return
	}
	if !h.authorizeAction(context, auth.ActionCreateReleaseRecord, packageName, uri.Version) {
		return
	}

	rawBody, readErr := io.ReadAll(context.Request.Body)
	if readErr != nil {
		response.WriteCatalogError(context, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": readErr.Error()}))
		return
	}
	release := h.acquireWriteLimiter(context)
	if release == nil {
		return
	}
	defer release()

	if h.CreateReleaseRecord == nil {
		response.WriteJSON(context, stdhttp.StatusCreated, catalog.ReleaseRecordResult{})
		return
	}
	payload, status, handlerErr := h.CreateReleaseRecord(context.Request, auth.PrincipalFromContext(context.Request.Context()), packageName, uri.Version, rawBody)
	if handlerErr != nil {
		response.WriteCatalogError(context, handlerErr)
		return
	}
	response.WriteJSON(context, status, payload)
}

func (h Handlers) authorizePackage(context *gin.Context, packageName string) bool {
	if h.AuthorizePackage == nil {
		return true
	}
	return h.AuthorizePackage(context.Writer, context.Request, packageName)
}

func (h Handlers) authorizeAction(context *gin.Context, action auth.Action, packageName string, version string) bool {
	if h.AuthorizeAction == nil {
		return true
	}
	return h.AuthorizeAction(context.Writer, context.Request, action, packageName, version)
}

func (h Handlers) acquireWriteLimiter(context *gin.Context) func() {
	if h.AcquireWriteLimiter == nil {
		return func() {}
	}
	release, err := h.AcquireWriteLimiter()
	if err != nil {
		response.WriteCatalogError(context, err)
		return nil
	}
	return release
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

func registerHeadCompatibleRoute(engine *gin.Engine, path string, handler gin.HandlerFunc) {
	engine.Match([]string{stdhttp.MethodGet, stdhttp.MethodHead}, path, handler)
}
