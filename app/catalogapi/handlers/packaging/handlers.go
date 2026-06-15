package packaginghandlers

import internalpackaginghttp "skill_registry/internal/packaging/http"

type AuthorizePackageFunc = internalpackaginghttp.AuthorizePackageFunc
type AcquireWriteLimiterFunc = internalpackaginghttp.AcquireWriteLimiterFunc
type DecodePrepareRequestFunc = internalpackaginghttp.DecodePrepareRequestFunc
type DecodeValidateRequestFunc = internalpackaginghttp.DecodeValidateRequestFunc
type DecodePackRequestFunc = internalpackaginghttp.DecodePackRequestFunc
type PrepareFunc = internalpackaginghttp.PrepareFunc
type ValidateFunc = internalpackaginghttp.ValidateFunc
type PackFunc = internalpackaginghttp.PackFunc
type PublishFunc = internalpackaginghttp.PublishFunc
type CleanupFunc = internalpackaginghttp.CleanupFunc
type DownloadBundleFunc = internalpackaginghttp.DownloadBundleFunc
type RecordMutationAuditFunc = internalpackaginghttp.RecordMutationAuditFunc
type Handlers = internalpackaginghttp.Handlers

func New(config Handlers) Handlers {
	return internalpackaginghttp.New(config)
}
