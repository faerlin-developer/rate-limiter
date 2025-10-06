package ratelimit

import "time"

type ObserveHooks[K comparable] struct {
	OnAllow func(key K, now time.Time)
	OnDeny  func(key K, err DeniedError)
}

// NewEmptyObserveHooks return hooks with no-op.
func NewEmptyObserveHooks[K comparable]() ObserveHooks[K] {
	return ObserveHooks[K]{
		OnAllow: func(key K, now time.Time) {},
		OnDeny:  func(key K, err DeniedError) {},
	}
}
