package commandcache

import (
	"context"
	"sync"
)

type InMemoryCommandCache struct {
	mu      *sync.Mutex
	storage map[string]struct{}
}

// NewInMemoryCommandCache - create new InMemoryCommandCache
func NewInMemoryCommandCache() *InMemoryCommandCache {
	return &InMemoryCommandCache{
		storage: make(map[string]struct{}),
	}
}

// Exists - check if commandID is already saved in Redis
//
// if it is, skip the command - it's already in execution (or finished)
func (s *InMemoryCommandCache) Exists(ctx context.Context, commandID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.storage[commandID]
	return ok, nil
}

// Save - store commandID in Redis
func (s *InMemoryCommandCache) Save(ctx context.Context, commandID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.storage[commandID] = struct{}{}
	return nil
}
