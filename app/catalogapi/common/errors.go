package common

import platformhttp "skill_registry/internal/platform/http"

var (
	CatalogErrorFromError = platformhttp.CatalogErrorFromError
	StatusFromError       = platformhttp.StatusFromError
	HashRequestPayload    = platformhttp.HashRequestPayload
)
