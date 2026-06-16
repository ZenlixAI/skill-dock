package http

import (
	"context"
	"io"
	stdhttp "net/http"

	catalog "skill_registry/internal/catalog/domain"
)

type RawBodyMutationConfig struct {
	Prepare             func(context.Context, []byte) *catalog.Error
	Authorize           func([]byte) bool
	AcquireWriteLimiter func() (func(), *catalog.Error)
	DefaultStatus       int
	DefaultPayload      any
	Execute             func(context.Context, []byte) (any, int, *catalog.Error)
}

func ExecuteRawBodyMutation(writer stdhttp.ResponseWriter, request *stdhttp.Request, config RawBodyMutationConfig) {
	rawBody, readErr := io.ReadAll(request.Body)
	if readErr != nil {
		WriteCatalogError(writer, request, catalog.NewError(422, "invalid_request", "request body is invalid JSON.", map[string]any{"error": readErr.Error()}))
		return
	}

	if config.Prepare != nil {
		if err := config.Prepare(request.Context(), rawBody); err != nil {
			WriteCatalogError(writer, request, err)
			return
		}
	}
	if config.Authorize != nil && !config.Authorize(rawBody) {
		return
	}

	release := func() {}
	if config.AcquireWriteLimiter != nil {
		limiterRelease, err := config.AcquireWriteLimiter()
		if err != nil {
			WriteCatalogError(writer, request, err)
			return
		}
		release = limiterRelease
	}
	defer release()

	if config.Execute == nil {
		status := config.DefaultStatus
		if status == 0 {
			status = stdhttp.StatusOK
		}
		WriteJSON(writer, status, config.DefaultPayload)
		return
	}

	payload, status, err := config.Execute(request.Context(), rawBody)
	if err != nil {
		WriteCatalogError(writer, request, err)
		return
	}
	WriteJSON(writer, status, payload)
}
