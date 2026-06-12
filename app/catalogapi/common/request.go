package common

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"skill_registry/business/catalog"
)

func DecodeJSONBody(request *http.Request, target any) *catalog.Error {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		if err == io.EOF {
			return catalog.NewError(422, "invalid_request", "request body is required.", nil)
		}
		return catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": err.Error()})
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return catalog.NewError(422, "invalid_request", "request body must contain a single JSON object.", nil)
	}

	return nil
}

func DecodeJSONBytes(raw []byte, target any) *catalog.Error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		if err == io.EOF {
			return catalog.NewError(422, "invalid_request", "request body is required.", nil)
		}
		return catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": err.Error()})
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return catalog.NewError(422, "invalid_request", "request body must contain a single JSON object.", nil)
	}

	return nil
}

func DecodeOptionalReleaseRecordInput(request *http.Request) (catalog.ReleaseRecordInput, *catalog.Error) {
	if request.Body == nil {
		return catalog.ReleaseRecordInput{}, nil
	}

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	var payload catalog.ReleaseRecordInput
	if err := decoder.Decode(&payload); err != nil {
		if err == io.EOF {
			return catalog.ReleaseRecordInput{}, nil
		}
		return catalog.ReleaseRecordInput{}, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": err.Error()})
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return catalog.ReleaseRecordInput{}, catalog.NewError(422, "invalid_request", "request body must contain a single JSON object.", nil)
	}

	return payload, nil
}

func DecodeOptionalReleaseRecordInputBytes(rawBody []byte) (catalog.ReleaseRecordInput, *catalog.Error) {
	if len(bytes.TrimSpace(rawBody)) == 0 {
		return catalog.ReleaseRecordInput{}, nil
	}

	decoder := json.NewDecoder(bytes.NewReader(rawBody))
	decoder.DisallowUnknownFields()

	var payload catalog.ReleaseRecordInput
	if err := decoder.Decode(&payload); err != nil {
		if err == io.EOF {
			return catalog.ReleaseRecordInput{}, nil
		}
		return catalog.ReleaseRecordInput{}, catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "request body is invalid JSON.", map[string]any{"error": err.Error()})
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return catalog.ReleaseRecordInput{}, catalog.NewError(http.StatusUnprocessableEntity, "invalid_request", "request body must contain a single JSON object.", nil)
	}
	return payload, nil
}
