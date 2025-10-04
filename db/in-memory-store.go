package db

import (
	lru "github.com/hashicorp/golang-lru/v2"
	"sync"
)

type Entry[V any] struct {
	value      V
	perKeyLock *sync.Mutex
}

// LRU cache
type InMemoryStore[K comparable, V any] struct {
	cache *lru.Cache[K, Entry[V]]
	mutex sync.RWMutex // zero value; ready to use
}

func NewInMemoryStore[K comparable, V any](capacity int) (*InMemoryStore[K, V], error) {

	cache, err := lru.New[K, Entry[V]](capacity)
	if err != nil {
		return nil, err
	}

	return &InMemoryStore[K, V]{
		cache: cache,
	}, nil
}

func (s *InMemoryStore[K, V]) Get(key K) (V, bool) {

	s.mutex.RLock()
	defer s.mutex.RUnlock()
	entry, ok := s.cache.Get(key)
	return entry.value, ok
}

func (s *InMemoryStore[K, V]) Put(key K, value V) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.cache.Contains(key) {
		entry, _ := s.cache.Get(key)
		entry.value = value
		s.cache.Add(key, entry)
	} else {
		entry := Entry[V]{
			value:      value,
			perKeyLock: &sync.Mutex{},
		}
		s.cache.Add(key, entry)
	}
}
