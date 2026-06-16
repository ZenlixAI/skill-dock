package http

import (
	"context"
	"encoding/json"
	"errors"
	stdhttp "net/http"
	"strings"
	"sync"
	"time"

	"skill_registry/app/catalogapi/audit"
	catalog "skill_registry/internal/catalog/domain"
	packagingapplication "skill_registry/internal/packaging/application"
	packaging "skill_registry/internal/packaging/domain"
	"skill_registry/internal/platform/auth"
	platformhttp "skill_registry/internal/platform/http"
	persist "skill_registry/internal/platform/persistence"
)

type WorkflowService = packagingapplication.WorkflowService

type WorkflowConfig struct {
	Service             WorkflowService
	Idempotency         persist.IdempotencyStore
	Audit               *audit.Recorder
	CacheTTL            time.Duration
	AllowAnonymousCache bool
}

type Workflow struct {
	service             WorkflowService
	idempotency         persist.IdempotencyStore
	audit               *audit.Recorder
	cache               *responseCache
	allowAnonymousCache bool
}

func NewWorkflow(config WorkflowConfig) *Workflow {
	cacheTTL := config.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 10 * time.Minute
	}
	return &Workflow{
		service:             config.Service,
		idempotency:         config.Idempotency,
		audit:               config.Audit,
		cache:               newResponseCache(cacheTTL),
		allowAnonymousCache: config.AllowAnonymousCache,
	}
}

func (w *Workflow) Prepare(ctx context.Context, principal auth.Principal, payload packaging.PrepareRequest) (any, int, *catalog.Error) {
	return executePackagingWorkflow(ctx, w, principal, "prepare", payload, true, func(payload packaging.PrepareRequest) *catalog.Error {
		if !strings.EqualFold(strings.TrimSpace(payload.Source.Type), "local") {
			return nil
		}
		return catalog.NewError(422, "unsupported_source_type", "source.type=local has been removed. Upload source archive content instead of sending a local filesystem path.", map[string]any{
			"sourceType":      payload.Source.Type,
			"suggestedAction": "Use source.type=upload_archive, git, or platform_authoring instead.",
		})
	}, func(payload packaging.PrepareRequest) (any, error) {
		return w.service.PrepareContext(ctx, payload)
	})
}

func (w *Workflow) Validate(ctx context.Context, principal auth.Principal, payload packaging.ValidateRequest) (any, int, *catalog.Error) {
	return executePackagingWorkflow(ctx, w, principal, "validate", payload, true, nil, func(payload packaging.ValidateRequest) (any, error) {
		return w.service.ValidateContext(ctx, payload)
	})
}

func (w *Workflow) Pack(ctx context.Context, principal auth.Principal, payload packaging.PackRequest) (any, int, *catalog.Error) {
	return executePackagingWorkflow(ctx, w, principal, "pack", payload, !payload.IncludeBundle, nil, func(payload packaging.PackRequest) (any, error) {
		return w.service.PackContext(ctx, payload)
	})
}

func (w *Workflow) DecodePrepareRequest(ctx context.Context, principal auth.Principal, rawBody []byte) (packaging.PrepareRequest, *catalog.Error) {
	return decodePackagingPayload[packaging.PrepareRequest](ctx, w, principal, "prepare", rawBody)
}

func (w *Workflow) DecodeValidateRequest(ctx context.Context, principal auth.Principal, rawBody []byte) (packaging.ValidateRequest, *catalog.Error) {
	return decodePackagingPayload[packaging.ValidateRequest](ctx, w, principal, "validate", rawBody)
}

func (w *Workflow) DecodePackRequest(ctx context.Context, principal auth.Principal, rawBody []byte) (packaging.PackRequest, *catalog.Error) {
	return decodePackagingPayload[packaging.PackRequest](ctx, w, principal, "pack", rawBody)
}

func decodePackagingPayload[T interface{ GetTarget() packaging.Target }](ctx context.Context, workflow *Workflow, principal auth.Principal, endpoint string, rawBody []byte) (T, *catalog.Error) {
	var payload T
	if err := platformhttp.DecodeJSONBytes(rawBody, &payload); err != nil {
		workflow.recordPackagingAudit(ctx, principal, endpoint, packaging.Target{}, err.Status, "failed", map[string]any{
			"errorCode": err.Code,
		})
		return payload, err
	}
	return payload, nil
}

func executePackagingWorkflow[T interface {
	GetIdempotencyKey() string
	GetTarget() packaging.Target
}](ctx context.Context, workflow *Workflow, principal auth.Principal, endpoint string, payload T, cacheAllowed bool, preflight func(T) *catalog.Error, run func(T) (any, error)) (any, int, *catalog.Error) {
	if workflow == nil || workflow.service == nil {
		return nil, stdhttp.StatusNotFound, nil
	}
	target := payload.GetTarget()
	if preflight != nil {
		if err := preflight(payload); err != nil {
			workflow.recordPackagingAudit(ctx, principal, endpoint, target, err.Status, "failed", map[string]any{
				"errorCode": err.Code,
			})
			return nil, err.Status, err
		}
	}

	requestHash, err := platformhttp.HashRequestPayload(payload)
	if err != nil {
		return nil, 0, catalog.NewError(500, "internal_error", "Internal server error.", nil)
	}

	idempotencyKey := strings.TrimSpace(payload.GetIdempotencyKey())
	if idempotencyKey == "" && workflow.allowAnonymousCache {
		idempotencyKey = requestHash
	}

	if cacheAllowed {
		if cached, cachedStatus, cacheErr, found := workflow.lookupCachedPackagingResponse(ctx, endpoint, principal.Role, idempotencyKey, requestHash); found {
			if cacheErr != nil {
				workflow.recordPackagingAudit(ctx, principal, endpoint, target, cacheErr.Status, "failed", map[string]any{
					"errorCode": cacheErr.Code,
				})
				return nil, cacheErr.Status, cacheErr
			}
			workflow.recordPackagingAudit(ctx, principal, endpoint, target, cachedStatus, "succeeded", map[string]any{
				"idempotentReplay": true,
			})
			return json.RawMessage(cached), cachedStatus, nil
		}
	}

	response, runErr := run(payload)
	if runErr != nil {
		catalogErr := platformhttp.CatalogErrorFromError(runErr)
		workflow.recordPackagingAudit(ctx, principal, endpoint, target, catalogErr.Status, "failed", map[string]any{
			"errorCode": catalogErr.Code,
		})
		return nil, platformhttp.StatusFromError(runErr), catalogErr
	}

	if idempotencyKey == "" || !cacheAllowed {
		var details map[string]any
		if !cacheAllowed {
			details = map[string]any{"idempotencyBypassed": true}
		}
		workflow.recordPackagingAudit(ctx, principal, endpoint, target, stdhttp.StatusOK, "succeeded", details)
		return response, stdhttp.StatusOK, nil
	}

	responseBody, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return nil, 0, catalog.NewError(500, "internal_error", "Internal server error.", nil)
	}
	responseBody, responseStatus := workflow.storeCachedPackagingResponse(ctx, endpoint, principal.Role, idempotencyKey, requestHash, stdhttp.StatusOK, responseBody)
	workflow.recordPackagingAudit(ctx, principal, endpoint, target, responseStatus, "succeeded", map[string]any{
		"idempotencyKey": idempotencyKey,
	})
	return json.RawMessage(responseBody), responseStatus, nil
}

func (w *Workflow) lookupCachedPackagingResponse(ctx context.Context, endpoint string, role string, idempotencyKey string, requestHash string) ([]byte, int, *catalog.Error, bool) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return nil, 0, nil, false
	}

	if w.idempotency != nil {
		record, err := w.idempotency.Lookup(ctx, endpoint, role, idempotencyKey)
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

	record, ok := w.cache.lookup(endpoint, role, idempotencyKey)
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

func (w *Workflow) storeCachedPackagingResponse(ctx context.Context, endpoint string, role string, idempotencyKey string, requestHash string, status int, body []byte) ([]byte, int) {
	record := persist.IdempotencyRecord{
		RequestHash:  requestHash,
		Status:       status,
		ResponseBody: append([]byte(nil), body...),
		CreatedAt:    time.Now().UTC(),
	}
	if w.idempotency != nil && strings.TrimSpace(idempotencyKey) != "" {
		_ = w.idempotency.DeleteExpired(ctx)
		saved, err := w.idempotency.Save(ctx, endpoint, role, strings.TrimSpace(idempotencyKey), record)
		if err == nil && saved {
			return append([]byte(nil), record.ResponseBody...), record.Status
		}
		if err == nil {
			existing, lookupErr := w.idempotency.Lookup(ctx, endpoint, role, strings.TrimSpace(idempotencyKey))
			if lookupErr == nil && existing.RequestHash == requestHash {
				return append([]byte(nil), existing.ResponseBody...), existing.Status
			}
		}
	}

	w.cache.store(endpoint, role, strings.TrimSpace(idempotencyKey), cachedResponse{
		requestHash: record.RequestHash,
		status:      record.Status,
		body:        append([]byte(nil), record.ResponseBody...),
		createdAt:   record.CreatedAt,
	})
	return append([]byte(nil), record.ResponseBody...), record.Status
}

func (w *Workflow) recordPackagingAudit(ctx context.Context, principal auth.Principal, endpoint string, target packaging.Target, status int, outcome string, details map[string]any) {
	if w == nil || w.audit == nil {
		return
	}
	_ = w.audit.RecordMutationForEndpoint(
		ctx,
		principal,
		stdhttp.MethodPost,
		"/api/v1/skill-packaging/"+endpoint,
		"packaging."+endpoint,
		target.PackageName,
		target.Version,
		status,
		outcome,
		details,
	)
}

type responseCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]cachedResponse
}

type cachedResponse struct {
	requestHash string
	status      int
	body        []byte
	createdAt   time.Time
}

func newResponseCache(ttl time.Duration) *responseCache {
	return &responseCache{
		ttl:     ttl,
		entries: make(map[string]cachedResponse),
	}
}

func (c *responseCache) lookup(endpoint string, role string, idempotencyKey string) (cachedResponse, bool) {
	if c == nil {
		return cachedResponse{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pruneExpiredLocked()

	record, ok := c.entries[endpoint+"\x00"+role+"\x00"+idempotencyKey]
	return record, ok
}

func (c *responseCache) store(endpoint string, role string, idempotencyKey string, record cachedResponse) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pruneExpiredLocked()
	c.entries[endpoint+"\x00"+role+"\x00"+idempotencyKey] = record
}

func (c *responseCache) pruneExpiredLocked() {
	if c == nil || c.ttl <= 0 {
		return
	}
	cutoff := time.Now().UTC().Add(-c.ttl)
	for key, record := range c.entries {
		if record.createdAt.Before(cutoff) {
			delete(c.entries, key)
		}
	}
}
