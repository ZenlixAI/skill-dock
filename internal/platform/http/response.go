package http

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	stdhttp "net/http"

	catalog "skill_registry/internal/catalog/domain"
	"skill_registry/internal/platform/auth"
)

func CatalogErrorFromError(err error) *catalog.Error {
	if err == nil {
		return nil
	}
	if catalogErr, ok := err.(*catalog.Error); ok {
		return catalogErr
	}
	return catalog.NewError(stdhttp.StatusInternalServerError, "internal_error", "Internal server error.", nil)
}

func StatusFromError(err error) int {
	catalogErr, ok := err.(*catalog.Error)
	if !ok {
		return stdhttp.StatusInternalServerError
	}
	return catalogErr.Status
}

func HashRequestPayload(payload any) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func DecodeJSONBody(request *stdhttp.Request, target any) *catalog.Error {
	return decodeJSON(request.Body, target, true)
}

func DecodeJSONBytes(raw []byte, target any) *catalog.Error {
	return decodeJSON(bytes.NewReader(raw), target, true)
}

func DecodeOptionalReleaseRecordInput(request *stdhttp.Request) (catalog.ReleaseRecordInput, *catalog.Error) {
	if request.Body == nil {
		return catalog.ReleaseRecordInput{}, nil
	}
	return decodeOptionalReleaseRecordInput(request.Body)
}

func DecodeOptionalReleaseRecordInputBytes(rawBody []byte) (catalog.ReleaseRecordInput, *catalog.Error) {
	if len(bytes.TrimSpace(rawBody)) == 0 {
		return catalog.ReleaseRecordInput{}, nil
	}
	return decodeOptionalReleaseRecordInput(bytes.NewReader(rawBody))
}

func WriteJSON(writer stdhttp.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}

func WriteJSONBytes(writer stdhttp.ResponseWriter, status int, payload []byte) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_, _ = writer.Write(payload)
}

func WriteCatalogError(writer stdhttp.ResponseWriter, request *stdhttp.Request, err error) {
	catalogErr, ok := err.(*catalog.Error)
	if !ok {
		catalogErr = catalog.NewError(stdhttp.StatusInternalServerError, "internal_error", "Internal server error.", nil)
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

func WriteAuthorizationError(writer stdhttp.ResponseWriter, request *stdhttp.Request, err *auth.AuthorizationError) {
	WriteCatalogError(writer, request, err.CatalogError())
}

func decodeJSON(reader io.Reader, target any, required bool) *catalog.Error {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		if err == io.EOF {
			if required {
				return catalog.NewError(stdhttp.StatusUnprocessableEntity, "invalid_request", "request body is required.", nil)
			}
			return nil
		}
		return catalog.NewError(stdhttp.StatusUnprocessableEntity, "invalid_request", "request body is invalid JSON.", map[string]any{"error": err.Error()})
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return catalog.NewError(stdhttp.StatusUnprocessableEntity, "invalid_request", "request body must contain a single JSON object.", nil)
	}

	return nil
}

func decodeOptionalReleaseRecordInput(reader io.Reader) (catalog.ReleaseRecordInput, *catalog.Error) {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()

	var payload catalog.ReleaseRecordInput
	if err := decoder.Decode(&payload); err != nil {
		if err == io.EOF {
			return catalog.ReleaseRecordInput{}, nil
		}
		return catalog.ReleaseRecordInput{}, catalog.NewError(stdhttp.StatusUnprocessableEntity, "invalid_request", "request body is invalid JSON.", map[string]any{"error": err.Error()})
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return catalog.ReleaseRecordInput{}, catalog.NewError(stdhttp.StatusUnprocessableEntity, "invalid_request", "request body must contain a single JSON object.", nil)
	}

	return payload, nil
}
