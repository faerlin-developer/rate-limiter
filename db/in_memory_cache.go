package db

import (
	lru "github.com/hashicorp/golang-lru/v2"
	"sync"
)

// Record wraps the value with a per-key lock.
type Record[V any] struct {
	value      V
	perKeyLock *sync.Mutex
}

// InMemoryCache is an in-memory implementation of the Cache interface.
type InMemoryCache[K comparable, V any] struct {
	cache *lru.Cache[K, Record[V]]
	mutex sync.RWMutex
}

func NewInMemoryCache[K comparable, V any](capacity int) (*InMemoryCache[K, V], error) {

	cache, err := lru.New[K, Record[V]](capacity)
	if err != nil {
		return nil, err
	}

	return &InMemoryCache[K, V]{
		cache: cache,
	}, nil
}

func (s *InMemoryCache[K, V]) Get(key K) (V, bool) {

	s.mutex.RLock()
	defer s.mutex.RUnlock()
	entry, ok := s.cache.Get(key)
	return entry.value, ok
}

func (s *InMemoryCache[K, V]) Put(key K, value V) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.cache.Contains(key) {
		// Update the inner value of the existing record
		record, _ := s.cache.Get(key)
		record.value = value
		s.cache.Add(key, record)
	} else {
		// Add new record into cache
		record := Record[V]{
			value:      value,
			perKeyLock: &sync.Mutex{},
		}
		s.cache.Add(key, record)
	}
}

func (s *InMemoryCache[K, V]) GetOrStore(key K, value V) (V, bool) {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Fast path: return the existing value
	record, ok := s.cache.Get(key)
	if ok {
		return record.value, false
	}

	// Slow path: add new record into cache
	record = Record[V]{
		value:      value,
		perKeyLock: &sync.Mutex{},
	}
	s.cache.Add(key, record)

	return value, true
}

func (s *InMemoryCache[K, V]) Contains(key K) bool {
	return s.cache.Contains(key)
}

func (s *InMemoryCache[K, V]) Lock(key K) {

	record, ok := s.cache.Get(key)
	if !ok {
		return
	}

	record.perKeyLock.Lock()
}

func (s *InMemoryCache[K, V]) Unlock(key K) {

	record, ok := s.cache.Get(key)
	if !ok {
		return
	}

	record.perKeyLock.Unlock()
}
