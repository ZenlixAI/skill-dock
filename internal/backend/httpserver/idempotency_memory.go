package httpserver

import (
	"context"
	"sync"

	persist "skill_registry/internal/platform/persistence"
)

type memoryIdempotencyStore struct {
	mu      sync.Mutex
	records map[string]persist.IdempotencyRecord
}

func newMemoryIdempotencyStore() persist.IdempotencyStore {
	return &memoryIdempotencyStore{records: map[string]persist.IdempotencyRecord{}}
}

func (s *memoryIdempotencyStore) Lookup(_ context.Context, endpoint string, role string, key string) (persist.IdempotencyRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.records[endpoint+"\x00"+role+"\x00"+key]
	if !ok {
		return persist.IdempotencyRecord{}, persist.ErrIdempotencyRecordNotFound
	}
	return record, nil
}

func (s *memoryIdempotencyStore) Save(_ context.Context, endpoint string, role string, key string, record persist.IdempotencyRecord) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cacheKey := endpoint + "\x00" + role + "\x00" + key
	if _, ok := s.records[cacheKey]; ok {
		return false, nil
	}
	s.records[cacheKey] = record
	return true, nil
}

func (s *memoryIdempotencyStore) DeleteExpired(context.Context) error {
	return nil
}
