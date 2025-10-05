package db

type Store[K comparable, V any] interface {
	Get(key K) (V, bool)
	Put(key K, value V)
}
