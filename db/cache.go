package db

type Cache[K comparable, V any] interface {
	Get(key K) (V, bool)
	Put(key K, value V)
	Contains(key K) bool
	GetOrStore(key K, value V) (V, bool)
	Lock(key K)
	Unlock(key K)
}
