// Package common provides the transport-level helpers shared across the catalog
// API modules: JSON encoding/decoding, query-parameter parsing, error
// translation, and request-scoped metadata propagation.
package common

import platformhttp "skill_registry/internal/platform/http"

var (
	ContextWithRequestMetadata = platformhttp.ContextWithRequestMetadata
	RequestIDFromContext       = platformhttp.RequestIDFromContext
	TraceIDFromContext         = platformhttp.TraceIDFromContext
)
