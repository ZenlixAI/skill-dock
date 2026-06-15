package common

import platformhttp "skill_registry/internal/platform/http"

var (
	WriteJSON               = platformhttp.WriteJSON
	WriteJSONBytes          = platformhttp.WriteJSONBytes
	WriteCatalogError       = platformhttp.WriteCatalogError
	WriteAuthorizationError = platformhttp.WriteAuthorizationError
)
