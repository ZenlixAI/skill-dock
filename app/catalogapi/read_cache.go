package catalogapi

import (
	"sync"
	"time"
)

type readCache struct {
	mu      sync.RWMutex
	entries map[string]readCacheEntry
}

type readCacheEntry struct {
	body      []byte
	expiresAt time.Time
}

func newReadCache() *readCache {
	return &readCache{entries: map[string]readCacheEntry{}}
}

func (c *readCache) lookup(key string) ([]byte, bool) {
	if c == nil || key == "" {
		return nil, false
	}

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil, false
	}
	return append([]byte(nil), entry.body...), true
}

func (c *readCache) store(key string, body []byte, ttl time.Duration) {
	if c == nil || key == "" || len(body) == 0 || ttl <= 0 {
		return
	}

	c.mu.Lock()
	c.entries[key] = readCacheEntry{
		body:      append([]byte(nil), body...),
		expiresAt: time.Now().Add(ttl),
	}
	c.mu.Unlock()
}
