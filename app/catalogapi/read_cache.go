package catalogapi

import (
	"time"

	platformhttp "skill_registry/internal/platform/http"
)

type readCache struct {
	*platformhttp.ReadCache
}

func newReadCache() *readCache {
	return &readCache{ReadCache: platformhttp.NewReadCache()}
}

func (c *readCache) lookup(key string) ([]byte, bool) {
	if c == nil {
		return nil, false
	}
	return c.ReadCache.Lookup(key)
}

func (c *readCache) store(key string, body []byte, ttl time.Duration) {
	if c == nil {
		return
	}
	c.ReadCache.Store(key, body, ttl)
}
