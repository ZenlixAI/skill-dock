package http

import (
	"testing"
	"time"
)

func TestReadCacheExpiresEntriesAfterTTL(t *testing.T) {
	cache := NewReadCache()
	cache.Store("skills?query=pdf", []byte(`{"items":[1]}`), 20*time.Millisecond)

	if _, ok := cache.Lookup("skills?query=pdf"); !ok {
		t.Fatal("expected cache hit before ttl")
	}

	time.Sleep(30 * time.Millisecond)
	if _, ok := cache.Lookup("skills?query=pdf"); ok {
		t.Fatal("expected cache miss after ttl")
	}
}

func TestReadCacheScopesByPrincipalKey(t *testing.T) {
	cache := NewReadCache()
	cache.Store("reader\x00@company.skill/a\x00skills?query=pdf", []byte(`{"items":[1]}`), time.Second)

	if _, ok := cache.Lookup("reader\x00@company.skill/b\x00skills?query=pdf"); ok {
		t.Fatal("expected scoped cache miss")
	}
}
