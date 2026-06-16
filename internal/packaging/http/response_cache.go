package http

import (
	"sync"
	"time"
)

type cachedResponse struct {
	requestHash string
	status      int
	body        []byte
	createdAt   time.Time
}

type responseCache struct {
	ttl     time.Duration
	mu      sync.Mutex
	records map[string]cachedResponse
}

func newResponseCache(ttl time.Duration) *responseCache {
	return &responseCache{
		ttl:     ttl,
		records: map[string]cachedResponse{},
	}
}

func (c *responseCache) lookup(endpoint string, role string, idempotencyKey string) (cachedResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pruneExpiredLocked()
	record, ok := c.records[cacheKey(endpoint, role, idempotencyKey)]
	if !ok {
		return cachedResponse{}, false
	}
	record.body = append([]byte(nil), record.body...)
	return record, true
}

func (c *responseCache) store(endpoint string, role string, idempotencyKey string, record cachedResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.pruneExpiredLocked()
	record.body = append([]byte(nil), record.body...)
	c.records[cacheKey(endpoint, role, idempotencyKey)] = record
}

func (c *responseCache) pruneExpiredLocked() {
	if c == nil || c.ttl <= 0 {
		return
	}

	cutoff := time.Now().Add(-c.ttl)
	for key, record := range c.records {
		if record.createdAt.Before(cutoff) {
			delete(c.records, key)
		}
	}
}

func cacheKey(endpoint string, role string, idempotencyKey string) string {
	return endpoint + "\x00" + role + "\x00" + idempotencyKey
}
