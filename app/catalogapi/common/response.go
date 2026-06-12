package common

import (
	"encoding/json"
	"net/http"

	"skill_registry/app/auth"
	"skill_registry/business/catalog"
)

func WriteJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}

func WriteJSONBytes(writer http.ResponseWriter, status int, payload []byte) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_, _ = writer.Write(payload)
}

func WriteCatalogError(writer http.ResponseWriter, request *http.Request, err error) {
	catalogErr, ok := err.(*catalog.Error)
	if !ok {
		catalogErr = catalog.NewError(500, "internal_error", "Internal server error.", nil)
	}

	requestID := RequestIDFromContext(request.Context())
	WriteJSON(writer, catalogErr.Status, map[string]any{
		"error": map[string]any{
			"code":      catalogErr.Code,
			"message":   catalogErr.Message,
			"details":   catalogErr.Details,
			"requestId": requestID,
		},
	})
}

func WriteAuthorizationError(writer http.ResponseWriter, request *http.Request, err *auth.AuthorizationError) {
	WriteCatalogError(writer, request, err.CatalogError())
}
