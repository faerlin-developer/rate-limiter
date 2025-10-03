package store

import (
	"fmt"
	lru "github.com/hashicorp/golang-lru/v2"
	"sync"
)

// LRU cache
type InMemoryStore[K comparable, V any] struct {
	cache *lru.Cache[K, V]
	mutex sync.RWMutex // zero value; ready to use
}

func NewInMemoryStore[K comparable, V any](capacity int) (*InMemoryStore[K, V], error) {

	cache, err := lru.New[K, V](capacity)
	if err != nil {
		return nil, err
	}

	return &InMemoryStore[K, V]{
		cache: cache,
	}, nil
}

func (s *InMemoryStore[K, V]) Get(key K) (V, error) {

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	value, ok := s.cache.Get(key)
	if !ok {
		var zero V
		return zero, fmt.Errorf("failed to find key: %s", key)
	}

	return value, nil
}

func (s *InMemoryStore[K, V]) Put(key K, value V) bool {

	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.cache.Add(key, value)
}
