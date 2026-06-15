package catalogapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"skill_registry/app/auth"
	"skill_registry/app/catalogapi/audit"
	"skill_registry/app/catalogapi/common"
	"skill_registry/business/catalog"
	"skill_registry/business/packaging"
	"skill_registry/foundation/persist"
	packagingapplication "skill_registry/internal/packaging/application"
)

type packagingService = packagingapplication.WorkflowService

type packagingModule struct {
	service             packagingService
	idempotency         persist.IdempotencyStore
	audit               *audit.Recorder
	cache               *packagingResponseCache
	allowAnonymousCache bool
}

func (m *packagingModule) prepare(ctx context.Context, principal auth.Principal, payload packaging.PrepareRequest) (any, int, *catalog.Error) {
	return executePackagingWorkflow(ctx, m, principal, "prepare", payload, true, func(payload packaging.PrepareRequest) *catalog.Error {
		if !strings.EqualFold(strings.TrimSpace(payload.Source.Type), "local") {
			return nil
		}
		return catalog.NewError(422, "unsupported_source_type", "source.type=local has been removed. Upload source archive content instead of sending a local filesystem path.", map[string]any{
			"sourceType":      payload.Source.Type,
			"suggestedAction": "Use source.type=upload_archive, git, or platform_authoring instead.",
		})
	}, func(payload packaging.PrepareRequest) (any, error) {
		return m.service.PrepareContext(ctx, payload)
	})
}

func (m *packagingModule) validate(ctx context.Context, principal auth.Principal, payload packaging.ValidateRequest) (any, int, *catalog.Error) {
	return executePackagingWorkflow(ctx, m, principal, "validate", payload, true, nil, func(payload packaging.ValidateRequest) (any, error) {
		return m.service.ValidateContext(ctx, payload)
	})
}

func (m *packagingModule) pack(ctx context.Context, principal auth.Principal, payload packaging.PackRequest) (any, int, *catalog.Error) {
	return executePackagingWorkflow(ctx, m, principal, "pack", payload, !payload.IncludeBundle, nil, func(payload packaging.PackRequest) (any, error) {
		return m.service.PackContext(ctx, payload)
	})
}

func (m *packagingModule) decodePrepareRequest(ctx context.Context, principal auth.Principal, rawBody []byte) (packaging.PrepareRequest, *catalog.Error) {
	return decodePackagingPayload[packaging.PrepareRequest](ctx, m, principal, "prepare", rawBody)
}

func (m *packagingModule) decodeValidateRequest(ctx context.Context, principal auth.Principal, rawBody []byte) (packaging.ValidateRequest, *catalog.Error) {
	return decodePackagingPayload[packaging.ValidateRequest](ctx, m, principal, "validate", rawBody)
}

func (m *packagingModule) decodePackRequest(ctx context.Context, principal auth.Principal, rawBody []byte) (packaging.PackRequest, *catalog.Error) {
	return decodePackagingPayload[packaging.PackRequest](ctx, m, principal, "pack", rawBody)
}

func decodePackagingPayload[T interface{ GetTarget() packaging.Target }](ctx context.Context, m *packagingModule, principal auth.Principal, endpoint string, rawBody []byte) (T, *catalog.Error) {
	var payload T
	if err := common.DecodeJSONBytes(rawBody, &payload); err != nil {
		m.recordPackagingAudit(ctx, principal, endpoint, packaging.Target{}, err.Status, "failed", map[string]any{
			"errorCode": err.Code,
		})
		return payload, err
	}
	return payload, nil
}

func executePackagingWorkflow[T interface {
	GetIdempotencyKey() string
	GetTarget() packaging.Target
}](ctx context.Context, m *packagingModule, principal auth.Principal, endpoint string, payload T, cacheAllowed bool, preflight func(T) *catalog.Error, run func(T) (any, error)) (any, int, *catalog.Error) {
	if m == nil || m.service == nil {
		return nil, http.StatusNotFound, nil
	}
	target := payload.GetTarget()
	if preflight != nil {
		if err := preflight(payload); err != nil {
			m.recordPackagingAudit(ctx, principal, endpoint, target, err.Status, "failed", map[string]any{
				"errorCode": err.Code,
			})
			return nil, err.Status, err
		}
	}

	requestHash, err := common.HashRequestPayload(payload)
	if err != nil {
		return nil, 0, catalog.NewError(500, "internal_error", "Internal server error.", nil)
	}

	idempotencyKey := strings.TrimSpace(payload.GetIdempotencyKey())
	if idempotencyKey == "" && m.allowAnonymousCache {
		idempotencyKey = requestHash
	}

	if cacheAllowed {
		if cached, cachedStatus, cacheErr, found := m.lookupCachedPackagingResponse(ctx, endpoint, principal.Role, idempotencyKey, requestHash); found {
			if cacheErr != nil {
				m.recordPackagingAudit(ctx, principal, endpoint, target, cacheErr.Status, "failed", map[string]any{
					"errorCode": cacheErr.Code,
				})
				return nil, cacheErr.Status, cacheErr
			}
			m.recordPackagingAudit(ctx, principal, endpoint, target, cachedStatus, "succeeded", map[string]any{
				"idempotentReplay": true,
			})
			return json.RawMessage(cached), cachedStatus, nil
		}
	}

	response, runErr := run(payload)
	if runErr != nil {
		catalogErr := common.CatalogErrorFromError(runErr)
		m.recordPackagingAudit(ctx, principal, endpoint, target, catalogErr.Status, "failed", map[string]any{
			"errorCode": catalogErr.Code,
		})
		return nil, common.StatusFromError(runErr), catalogErr
	}

	if idempotencyKey == "" || !cacheAllowed {
		var details map[string]any
		if !cacheAllowed {
			details = map[string]any{"idempotencyBypassed": true}
		}
		m.recordPackagingAudit(ctx, principal, endpoint, target, http.StatusOK, "succeeded", details)
		return response, http.StatusOK, nil
	}

	responseBody, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		return nil, 0, catalog.NewError(500, "internal_error", "Internal server error.", nil)
	}
	responseBody, responseStatus := m.storeCachedPackagingResponse(ctx, endpoint, principal.Role, idempotencyKey, requestHash, http.StatusOK, responseBody)
	m.recordPackagingAudit(ctx, principal, endpoint, target, responseStatus, "succeeded", map[string]any{
		"idempotencyKey": idempotencyKey,
	})
	return json.RawMessage(responseBody), responseStatus, nil
}

func (m *packagingModule) lookupCachedPackagingResponse(ctx context.Context, endpoint string, role string, idempotencyKey string, requestHash string) ([]byte, int, *catalog.Error, bool) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return nil, 0, nil, false
	}

	if m.idempotency != nil {
		record, err := m.idempotency.Lookup(ctx, endpoint, role, idempotencyKey)
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

	record, ok := m.cache.lookup(endpoint, role, idempotencyKey)
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

func (m *packagingModule) storeCachedPackagingResponse(ctx context.Context, endpoint string, role string, idempotencyKey string, requestHash string, status int, body []byte) ([]byte, int) {
	record := persist.IdempotencyRecord{
		RequestHash:  requestHash,
		Status:       status,
		ResponseBody: append([]byte(nil), body...),
		CreatedAt:    time.Now().UTC(),
	}
	if m.idempotency != nil && strings.TrimSpace(idempotencyKey) != "" {
		_ = m.idempotency.DeleteExpired(ctx)
		saved, err := m.idempotency.Save(ctx, endpoint, role, strings.TrimSpace(idempotencyKey), record)
		if err == nil && saved {
			return append([]byte(nil), record.ResponseBody...), record.Status
		}
		if err == nil {
			existing, lookupErr := m.idempotency.Lookup(ctx, endpoint, role, strings.TrimSpace(idempotencyKey))
			if lookupErr == nil && existing.RequestHash == requestHash {
				return append([]byte(nil), existing.ResponseBody...), existing.Status
			}
		}
	}

	m.cache.store(endpoint, role, strings.TrimSpace(idempotencyKey), cachedPackagingResponse{
		requestHash: record.RequestHash,
		status:      record.Status,
		body:        append([]byte(nil), record.ResponseBody...),
		createdAt:   record.CreatedAt,
	})
	return append([]byte(nil), record.ResponseBody...), record.Status
}

func (m *packagingModule) recordPackagingAudit(ctx context.Context, principal auth.Principal, endpoint string, target packaging.Target, status int, outcome string, details map[string]any) {
	if m == nil || m.audit == nil {
		return
	}
	_ = m.audit.RecordMutationForEndpoint(
		ctx,
		principal,
		http.MethodPost,
		"/api/v1/skill-packaging/"+endpoint,
		"packaging."+endpoint,
		target.PackageName,
		target.Version,
		status,
		outcome,
		details,
	)
}

type packagingResponseCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[string]cachedPackagingResponse
}

type cachedPackagingResponse struct {
	requestHash string
	status      int
	body        []byte
	createdAt   time.Time
}

func newPackagingResponseCache(ttl time.Duration) *packagingResponseCache {
	return &packagingResponseCache{
		ttl:     ttl,
		entries: make(map[string]cachedPackagingResponse),
	}
}

func (c *packagingResponseCache) lookup(endpoint string, role string, idempotencyKey string) (cachedPackagingResponse, bool) {
	if c == nil {
		return cachedPackagingResponse{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pruneExpiredLocked()

	record, ok := c.entries[endpoint+"\x00"+role+"\x00"+idempotencyKey]
	return record, ok
}

func (c *packagingResponseCache) store(endpoint string, role string, idempotencyKey string, record cachedPackagingResponse) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pruneExpiredLocked()
	c.entries[endpoint+"\x00"+role+"\x00"+idempotencyKey] = record
}

func (c *packagingResponseCache) pruneExpiredLocked() {
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
