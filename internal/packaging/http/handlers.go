package http

import (
	"context"
	"io"
	stdhttp "net/http"

	"github.com/gin-gonic/gin"

	catalog "skill_registry/internal/catalog/domain"
	packaging "skill_registry/internal/packaging/domain"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
)

type AuthorizePackageFunc func(stdhttp.ResponseWriter, *stdhttp.Request, string) bool
type AcquireWriteLimiterFunc func() (func(), *catalog.Error)
type DecodeImportReviewRequestFunc func(context.Context, auth.Principal, []byte) (packaging.ImportReviewRequest, *catalog.Error)
type ImportReviewFunc func(context.Context, auth.Principal, packaging.ImportReviewRequest) (any, int, *catalog.Error)
type PublishFunc func(context.Context, packaging.PublishRequest) (packaging.PublishResponse, error)
type RecordMutationAuditFunc func(request *stdhttp.Request, action string, packageName string, version string, status int, outcome string, details map[string]any)

type Handlers struct {
	HasPackaging              func() bool
	AuthorizePackage          AuthorizePackageFunc
	AcquireWriteLimiter       AcquireWriteLimiterFunc
	DecodeImportReviewRequest DecodeImportReviewRequestFunc
	ImportReview              ImportReviewFunc
	Publish                   PublishFunc
	RecordMutationAudit       RecordMutationAuditFunc
}

func New(config Handlers) Handlers {
	return config
}

func (h Handlers) ImportReviewHandler(c *gin.Context) {
	handlePackagingWorkflow(
		h,
		c,
		func(ctx context.Context, principal auth.Principal, rawBody []byte) (packaging.ImportReviewRequest, *catalog.Error) {
			return h.DecodeImportReviewRequest(ctx, principal, rawBody)
		},
		func(ctx context.Context, principal auth.Principal, payload packaging.ImportReviewRequest) (any, int, *catalog.Error) {
			return h.ImportReview(ctx, principal, payload)
		},
	)
}

func (h Handlers) PublishHandler(c *gin.Context) {
	if !h.packagingAvailable() {
		c.Status(stdhttp.StatusNotFound)
		return
	}

	var payload packaging.PublishRequest
	if err := platformhttp.DecodeJSONBody(c.Request, &payload); err != nil {
		h.recordMutationAudit(c.Request, "packaging.publish", "", "", err.Status, "failed", map[string]any{"errorCode": err.Code})
		platformhttp.WriteCatalogError(c.Writer, c.Request, err)
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
		platformhttp.WriteJSON(c.Writer, stdhttp.StatusOK, packaging.PublishResponse{})
		return
	}
	result, err := h.Publish(c.Request.Context(), payload)
	if err != nil {
		catalogErr := platformhttp.CatalogErrorFromError(err)
		h.recordMutationAudit(c.Request, "packaging.publish", payload.Target.PackageName, payload.Target.Version, catalogErr.Status, "failed", map[string]any{"errorCode": catalogErr.Code})
		platformhttp.WriteCatalogError(c.Writer, c.Request, catalogErr)
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
	platformhttp.WriteJSON(c.Writer, status, result)
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
		platformhttp.WriteCatalogError(context.Writer, context.Request, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": readErr.Error()}))
		return
	}

	principal := auth.PrincipalFromContext(context.Request.Context())
	payload, err := decode(context.Request.Context(), principal, rawBody)
	if err != nil {
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
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
		platformhttp.WriteCatalogError(context.Writer, context.Request, execErr)
		return
	}
	platformhttp.WriteJSON(context.Writer, status, result)
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
		platformhttp.WriteCatalogError(context.Writer, context.Request, err)
		return nil
	}
	return release
}

func (h Handlers) recordMutationAudit(request *stdhttp.Request, action string, packageName string, version string, status int, outcome string, details map[string]any) {
	if h.RecordMutationAudit != nil {
		h.RecordMutationAudit(request, action, packageName, version, status, outcome, details)
	}
}
