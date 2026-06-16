package http

import (
	"context"
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"strings"
	"time"

	catalog "skill_registry/internal/catalog/domain"
	packagingapplication "skill_registry/internal/packaging/application"
	packaging "skill_registry/internal/packaging/domain"
	platformaudit "skill_registry/internal/platform/audit"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
	persist "skill_registry/internal/platform/persistence"
)

type ReviewService = packagingapplication.ReviewService

type OneShotWorkflowConfig struct {
	ReviewService       ReviewService
	Idempotency         persist.IdempotencyStore
	Audit               *platformaudit.Recorder
	CacheTTL            time.Duration
	AllowAnonymousCache bool
}

type OneShotWorkflow struct {
	reviewService       ReviewService
	idempotency         persist.IdempotencyStore
	audit               *platformaudit.Recorder
	cache               *responseCache
	allowAnonymousCache bool
}

func NewOneShotWorkflow(config OneShotWorkflowConfig) *OneShotWorkflow {
	cacheTTL := config.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 10 * time.Minute
	}
	return &OneShotWorkflow{
		reviewService:       config.ReviewService,
		idempotency:         config.Idempotency,
		audit:               config.Audit,
		cache:               newResponseCache(cacheTTL),
		allowAnonymousCache: config.AllowAnonymousCache,
	}
}

func (w *OneShotWorkflow) DecodeImportReviewRequest(ctx context.Context, principal auth.Principal, rawBody []byte) (packaging.ImportReviewRequest, *catalog.Error) {
	var payload packaging.ImportReviewRequest
	if err := platformhttp.DecodeJSONBytes(rawBody, &payload); err != nil {
		w.recordImportReviewAudit(ctx, principal, packaging.Target{}, err.Status, "failed", map[string]any{
			"errorCode": err.Code,
		})
		return payload, err
	}
	return payload, nil
}

func (w *OneShotWorkflow) ImportReview(ctx context.Context, principal auth.Principal, payload packaging.ImportReviewRequest) (any, int, *catalog.Error) {
	if w == nil || w.reviewService == nil {
		return nil, stdhttp.StatusNotFound, nil
	}

	requestHash, err := platformhttp.HashRequestPayload(payload)
	if err != nil {
		return nil, 0, catalog.NewError(500, "internal_error", "Internal server error.", nil)
	}

	idempotencyKey := strings.TrimSpace(payload.GetIdempotencyKey())
	if idempotencyKey == "" && w.allowAnonymousCache {
		idempotencyKey = requestHash
	}
	if cached, cachedStatus, cacheErr, found := w.lookupCachedImportReviewResponse(ctx, principal.Role, idempotencyKey, requestHash); found {
		if cacheErr != nil {
			w.recordImportReviewAudit(ctx, principal, payload.Target, cacheErr.Status, "failed", map[string]any{
				"errorCode": cacheErr.Code,
			})
			return nil, cacheErr.Status, cacheErr
		}
		w.recordImportReviewAudit(ctx, principal, payload.Target, cachedStatus, "succeeded", map[string]any{
			"idempotentReplay": true,
		})
		return json.RawMessage(cached), cachedStatus, nil
	}

	response, err := w.reviewService.ReviewContext(ctx, payload)
	if err != nil {
		catalogErr := platformhttp.CatalogErrorFromError(err)
		w.recordImportReviewAudit(ctx, principal, payload.Target, catalogErr.Status, "failed", map[string]any{
			"errorCode": catalogErr.Code,
		})
		return nil, platformhttp.StatusFromError(err), catalogErr
	}

	if idempotencyKey == "" {
		w.recordImportReviewAudit(ctx, principal, payload.Target, stdhttp.StatusOK, "succeeded", nil)
		return response, stdhttp.StatusOK, nil
	}

	responseBody, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return nil, 0, catalog.NewError(500, "internal_error", "Internal server error.", nil)
	}
	responseBody, responseStatus := w.storeCachedImportReviewResponse(ctx, principal.Role, idempotencyKey, requestHash, stdhttp.StatusOK, responseBody)
	w.recordImportReviewAudit(ctx, principal, payload.Target, responseStatus, "succeeded", map[string]any{
		"idempotencyKey": idempotencyKey,
	})
	return json.RawMessage(responseBody), responseStatus, nil
}

func (w *OneShotWorkflow) lookupCachedImportReviewResponse(ctx context.Context, role string, idempotencyKey string, requestHash string) ([]byte, int, *catalog.Error, bool) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return nil, 0, nil, false
	}

	if w.idempotency != nil {
		record, err := w.idempotency.Lookup(ctx, "import-review", role, idempotencyKey)
		switch {
		case err == nil:
			if record.RequestHash != requestHash {
				return nil, 0, catalog.NewError(409, "idempotency_key_conflict", "idempotencyKey cannot be reused with a different request body.", map[string]any{
					"idempotencyKey": idempotencyKey,
				}), true
			}
			return append([]byte(nil), record.ResponseBody...), record.Status, nil, true
		case !errors.Is(err, persist.ErrIdempotencyRecordNotFound):
			return nil, 0, catalog.NewError(500, "internal_error", "Internal server error.", nil), true
		}
	}

	record, ok := w.cache.lookup("import-review", role, idempotencyKey)
	if !ok {
		return nil, 0, nil, false
	}
	if record.requestHash != requestHash {
		return nil, 0, catalog.NewError(409, "idempotency_key_conflict", "idempotencyKey cannot be reused with a different request body.", map[string]any{
			"idempotencyKey": idempotencyKey,
		}), true
	}
	return append([]byte(nil), record.body...), record.status, nil, true
}

func (w *OneShotWorkflow) storeCachedImportReviewResponse(ctx context.Context, role string, idempotencyKey string, requestHash string, status int, body []byte) ([]byte, int) {
	record := persist.IdempotencyRecord{
		RequestHash:  requestHash,
		Status:       status,
		ResponseBody: append([]byte(nil), body...),
		CreatedAt:    time.Now().UTC(),
	}
	if w.idempotency != nil && strings.TrimSpace(idempotencyKey) != "" {
		_ = w.idempotency.DeleteExpired(ctx)
		saved, err := w.idempotency.Save(ctx, "import-review", role, strings.TrimSpace(idempotencyKey), record)
		if err == nil && saved {
			return append([]byte(nil), record.ResponseBody...), record.Status
		}
		if err == nil {
			existing, lookupErr := w.idempotency.Lookup(ctx, "import-review", role, strings.TrimSpace(idempotencyKey))
			if lookupErr == nil && existing.RequestHash == requestHash {
				return append([]byte(nil), existing.ResponseBody...), existing.Status
			}
		}
	}

	w.cache.store("import-review", role, strings.TrimSpace(idempotencyKey), cachedResponse{
		requestHash: record.RequestHash,
		status:      record.Status,
		body:        append([]byte(nil), record.ResponseBody...),
		createdAt:   record.CreatedAt,
	})
	return append([]byte(nil), record.ResponseBody...), record.Status
}

func (w *OneShotWorkflow) recordImportReviewAudit(ctx context.Context, principal auth.Principal, target packaging.Target, status int, outcome string, details map[string]any) {
	if w == nil || w.audit == nil {
		return
	}
	_ = w.audit.RecordMutationForEndpoint(
		ctx,
		principal,
		stdhttp.MethodPost,
		"/api/v1/skill-packaging/import-review",
		"packaging.import-review",
		target.PackageName,
		target.Version,
		status,
		outcome,
		details,
	)
}
