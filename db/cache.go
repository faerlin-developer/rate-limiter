package db

// Cache is the interface for the rate limiter
type Cache[K comparable, V any] interface {

	// Get returns the value for the given key
	Get(key K) (V, bool)

	// Put stores the given key-value pair
	Put(key K, value V)

	// Contains returns true when key is present in the cache
	Contains(key K) bool

	// GetOrStore retrieves the value for key if present; otherwise inserts the given value and returns it.
	// If given value was inserted, return true for the second return value; otherwise return false.
	GetOrStore(key K, value V) (V, bool)

	// Lock acquire a lock on the given key
	Lock(key K)

	// Unlock releases the lock on the given key
	Unlock(key K)
}
