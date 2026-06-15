package http

import (
	"context"
	"io"
	stdhttp "net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/common"
	"skill_registry/app/catalogapi/response"
	"skill_registry/business/catalog"
	"skill_registry/business/packaging"
)

type AuthorizePackageFunc func(stdhttp.ResponseWriter, *stdhttp.Request, string) bool
type AcquireWriteLimiterFunc func() (func(), *catalog.Error)
type DecodePrepareRequestFunc func(context.Context, auth.Principal, []byte) (packaging.PrepareRequest, *catalog.Error)
type DecodeValidateRequestFunc func(context.Context, auth.Principal, []byte) (packaging.ValidateRequest, *catalog.Error)
type DecodePackRequestFunc func(context.Context, auth.Principal, []byte) (packaging.PackRequest, *catalog.Error)
type PrepareFunc func(context.Context, auth.Principal, packaging.PrepareRequest) (any, int, *catalog.Error)
type ValidateFunc func(context.Context, auth.Principal, packaging.ValidateRequest) (any, int, *catalog.Error)
type PackFunc func(context.Context, auth.Principal, packaging.PackRequest) (any, int, *catalog.Error)
type PublishFunc func(context.Context, packaging.PublishRequest) (packaging.PublishResponse, error)
type CleanupFunc func(context.Context, auth.Principal, packaging.CleanupRequest) (packaging.CleanupResponse, int, *catalog.Error)
type DownloadBundleFunc func(context.Context, auth.Principal, string, string, string) (string, *catalog.Error)
type RecordMutationAuditFunc func(request *stdhttp.Request, action string, packageName string, version string, status int, outcome string, details map[string]any)

type Handlers struct {
	HasPackaging          func() bool
	AuthorizePackage      AuthorizePackageFunc
	AcquireWriteLimiter   AcquireWriteLimiterFunc
	DecodePrepareRequest  DecodePrepareRequestFunc
	DecodeValidateRequest DecodeValidateRequestFunc
	DecodePackRequest     DecodePackRequestFunc
	Prepare               PrepareFunc
	Validate              ValidateFunc
	Pack                  PackFunc
	Publish               PublishFunc
	Cleanup               CleanupFunc
	DownloadBundle        DownloadBundleFunc
	RecordMutationAudit   RecordMutationAuditFunc
}

func New(config Handlers) Handlers {
	return config
}

func (h Handlers) PrepareHandler(c *gin.Context) {
	handlePackagingWorkflow(
		h,
		c,
		func(ctx context.Context, principal auth.Principal, rawBody []byte) (packaging.PrepareRequest, *catalog.Error) {
			return h.DecodePrepareRequest(ctx, principal, rawBody)
		},
		func(ctx context.Context, principal auth.Principal, payload packaging.PrepareRequest) (any, int, *catalog.Error) {
			return h.Prepare(ctx, principal, payload)
		},
	)
}

func (h Handlers) ValidateHandler(c *gin.Context) {
	handlePackagingWorkflow(
		h,
		c,
		func(ctx context.Context, principal auth.Principal, rawBody []byte) (packaging.ValidateRequest, *catalog.Error) {
			return h.DecodeValidateRequest(ctx, principal, rawBody)
		},
		func(ctx context.Context, principal auth.Principal, payload packaging.ValidateRequest) (any, int, *catalog.Error) {
			return h.Validate(ctx, principal, payload)
		},
	)
}

func (h Handlers) PackHandler(c *gin.Context) {
	handlePackagingWorkflow(
		h,
		c,
		func(ctx context.Context, principal auth.Principal, rawBody []byte) (packaging.PackRequest, *catalog.Error) {
			return h.DecodePackRequest(ctx, principal, rawBody)
		},
		func(ctx context.Context, principal auth.Principal, payload packaging.PackRequest) (any, int, *catalog.Error) {
			return h.Pack(ctx, principal, payload)
		},
	)
}

func (h Handlers) PublishHandler(c *gin.Context) {
	if !h.packagingAvailable() {
		c.Status(stdhttp.StatusNotFound)
		return
	}

	var payload packaging.PublishRequest
	if err := common.DecodeJSONBody(c.Request, &payload); err != nil {
		h.recordMutationAudit(c.Request, "packaging.publish", "", "", err.Status, "failed", map[string]any{"errorCode": err.Code})
		response.WriteCatalogError(c, err)
		return
	}
	if !h.authorizePackage(c, payload.Target.PackageName) {
		return
	}
	release := h.acquireWriteLimiter(c)
	if release == nil {
		return
	}
	defer release()

	if h.Publish == nil {
		response.WriteJSON(c, stdhttp.StatusOK, packaging.PublishResponse{})
		return
	}
	result, err := h.Publish(c.Request.Context(), payload)
	if err != nil {
		catalogErr := common.CatalogErrorFromError(err)
		h.recordMutationAudit(c.Request, "packaging.publish", payload.Target.PackageName, payload.Target.Version, catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
		response.WriteCatalogError(c, catalogErr)
		return
	}

	status := stdhttp.StatusOK
	if result.Created {
		status = stdhttp.StatusCreated
	}
	h.recordMutationAudit(c.Request, "packaging.publish", payload.Target.PackageName, payload.Target.Version, status, "succeeded", map[string]any{
		"publishedToRegistry": result.PublishedToRegistry,
		"created":             result.Created,
		"idempotentReplay":    result.IdempotentReplay,
	})
	response.WriteJSON(c, status, result)
}

func (h Handlers) CleanupHandler(c *gin.Context) {
	if !h.packagingAvailable() {
		c.Status(stdhttp.StatusNotFound)
		return
	}

	var payload packaging.CleanupRequest
	if err := common.DecodeJSONBody(c.Request, &payload); err != nil {
		h.recordMutationAudit(c.Request, "packaging.cleanup", "", "", err.Status, "failed", map[string]any{"errorCode": err.Code})
		response.WriteCatalogError(c, err)
		return
	}

	if h.Cleanup == nil {
		response.WriteJSON(c, stdhttp.StatusOK, packaging.CleanupResponse{})
		return
	}
	result, status, err := h.Cleanup(c.Request.Context(), auth.PrincipalFromContext(c.Request.Context()), payload)
	if err != nil {
		response.WriteCatalogError(c, err)
		return
	}
	response.WriteJSON(c, status, result)
}

func (h Handlers) BundleHandler(c *gin.Context) {
	if !h.packagingAvailable() {
		c.Status(stdhttp.StatusNotFound)
		return
	}

	query, err := BindBundleDownloadQuery(c)
	if err != nil {
		response.WriteCatalogError(c, err)
		return
	}
	packageName := strings.TrimSpace(query.PackageName)
	if !h.authorizePackage(c, packageName) {
		return
	}

	if h.DownloadBundle == nil {
		response.WriteCatalogError(c, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil))
		return
	}
	bundlePath, bundleErr := h.DownloadBundle(
		c.Request.Context(),
		auth.PrincipalFromContext(c.Request.Context()),
		strings.TrimSpace(query.WorkspacePath),
		packageName,
		strings.TrimSpace(query.Version),
	)
	if bundleErr != nil {
		response.WriteCatalogError(c, bundleErr)
		return
	}

	bundleFile, openErr := os.Open(bundlePath)
	if openErr != nil {
		response.WriteCatalogError(c, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil))
		return
	}
	defer bundleFile.Close()

	info, statErr := bundleFile.Stat()
	if statErr != nil || info.IsDir() {
		response.WriteCatalogError(c, catalog.NewError(500, "internal_error", "Failed to load packaging bundle.", nil))
		return
	}

	c.Writer.Header().Set("Content-Type", "application/octet-stream")
	stdhttp.ServeContent(c.Writer, c.Request, filepath.Base(bundlePath), info.ModTime(), bundleFile)
}

func handlePackagingWorkflow[T interface {
	GetIdempotencyKey() string
	GetTarget() packaging.Target
}](h Handlers, context *gin.Context, decode func(context.Context, auth.Principal, []byte) (T, *catalog.Error), execute func(context.Context, auth.Principal, T) (any, int, *catalog.Error)) {
	if !h.packagingAvailable() {
		context.Status(stdhttp.StatusNotFound)
		return
	}

	rawBody, readErr := io.ReadAll(context.Request.Body)
	if readErr != nil {
		response.WriteCatalogError(context, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": readErr.Error()}))
		return
	}

	principal := auth.PrincipalFromContext(context.Request.Context())
	payload, err := decode(context.Request.Context(), principal, rawBody)
	if err != nil {
		response.WriteCatalogError(context, err)
		return
	}
	if packageName := payload.GetTarget().PackageName; packageName != "" {
		if !h.authorizePackage(context, packageName) {
			return
		}
	}
	release := h.acquireWriteLimiter(context)
	if release == nil {
		return
	}
	defer release()

	result, status, execErr := execute(context.Request.Context(), principal, payload)
	if execErr != nil {
		response.WriteCatalogError(context, execErr)
		return
	}
	response.WriteJSON(context, status, result)
}

func (h Handlers) packagingAvailable() bool {
	if h.HasPackaging == nil {
		return false
	}
	return h.HasPackaging()
}

func (h Handlers) authorizePackage(context *gin.Context, packageName string) bool {
	if h.AuthorizePackage == nil {
		return true
	}
	return h.AuthorizePackage(context.Writer, context.Request, packageName)
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

func (h Handlers) recordMutationAudit(request *stdhttp.Request, action string, packageName string, version string, status int, outcome string, details map[string]any) {
	if h.RecordMutationAudit != nil {
		h.RecordMutationAudit(request, action, packageName, version, status, outcome, details)
	}
}
