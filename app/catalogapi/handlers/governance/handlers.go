package governancehandlers

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"skill_registry/app/auth"
	transportbinding "skill_registry/app/catalogapi/binding"
	"skill_registry/app/catalogapi/response"
	"skill_registry/business/catalog"
)

type AuthorizePackageFunc func(http.ResponseWriter, *http.Request, string) bool
type AuthorizeActionFunc func(http.ResponseWriter, *http.Request, auth.Action, string, string) bool
type AcquireWriteLimiterFunc func() (func(), *catalog.Error)
type CreateReleaseRecordFunc func(ctx any, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error)
type ApplySkillGovernanceChangeFunc func(ctx any, principal auth.Principal, packageName string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error)
type ApplyVersionGovernanceChangeFunc func(ctx any, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error)

type Handlers struct {
	AuthorizePackage           AuthorizePackageFunc
	AuthorizeAction            AuthorizeActionFunc
	AcquireWriteLimiter        AcquireWriteLimiterFunc
	CreateReleaseRecord        func(request *http.Request, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.ReleaseRecordResult, int, *catalog.Error)
	ApplySkillGovernanceChange func(request *http.Request, principal auth.Principal, packageName string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error)
	ApplyVersionGovernance     func(request *http.Request, principal auth.Principal, packageName string, versionPath string, rawBody []byte) (catalog.GovernanceChangeResponse, int, *catalog.Error)
}

func New(config Handlers) Handlers {
	return config
}

func (h Handlers) SkillGovernance(context *gin.Context) {
	uri, err := transportbinding.BindPackageURI(context)
	if err != nil {
		response.WriteCatalogError(context, err)
		return
	}
	packageName, packageErr := transportbinding.DecodePackageName(uri.Package)
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
		response.WriteJSON(context, http.StatusCreated, catalog.GovernanceChangeResponse{})
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
	uri, err := transportbinding.BindVersionURI(context)
	if err != nil {
		response.WriteCatalogError(context, err)
		return
	}
	packageName, packageErr := transportbinding.DecodePackageName(uri.Package)
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
		response.WriteJSON(context, http.StatusCreated, catalog.GovernanceChangeResponse{})
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
	uri, err := transportbinding.BindVersionURI(context)
	if err != nil {
		response.WriteCatalogError(context, err)
		return
	}
	packageName, packageErr := transportbinding.DecodePackageName(uri.Package)
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
		response.WriteJSON(context, http.StatusCreated, catalog.ReleaseRecordResult{})
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
