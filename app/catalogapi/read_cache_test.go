package catalogapi

import (
	"testing"
	"time"
)

func TestReadCacheExpiresEntriesAfterTTL(t *testing.T) {
	cache := newReadCache()
	cache.store("skills?query=pdf", []byte(`{"items":[1]}`), 20*time.Millisecond)

	if _, ok := cache.lookup("skills?query=pdf"); !ok {
		t.Fatal("expected cache hit before ttl")
	}

	time.Sleep(30 * time.Millisecond)
	if _, ok := cache.lookup("skills?query=pdf"); ok {
		t.Fatal("expected cache miss after ttl")
	}
}

func TestReadCacheScopesByPrincipalKey(t *testing.T) {
	cache := newReadCache()
	cache.store("reader\x00@company.skill/a\x00skills?query=pdf", []byte(`{"items":[1]}`), time.Second)

	if _, ok := cache.lookup("reader\x00@company.skill/b\x00skills?query=pdf"); ok {
		t.Fatal("expected scoped cache miss")
	}
}
