package ratelimit

import (
	"fmt"
	"github.com/faerlin-developer/rate-limiter.git/clock"
	"github.com/faerlin-developer/rate-limiter.git/db"
	"time"
)

const (
	DefaultTokensPerSecond = 10
	DefaultBucketCapacity  = 10
	DefaultCacheCapacity   = 100
)

type Option func(limiter *TBLimiter) error
type Clock clock.Clock
type TBCache db.Cache[string, Bucket]

func WithTokensPerSecond(tokensPerSecond int) Option {
	return func(l *TBLimiter) error {
		if tokensPerSecond <= 0 {
			return fmt.Errorf("tokensPerSecond must be greater than 0")
		}
		l.refillInterval = time.Second / time.Duration(tokensPerSecond)
		return nil
	}
}

func WithCache(cache TBCache) Option {
	return func(l *TBLimiter) error {
		l.cache = cache
		return nil
	}
}

func WithBucketCapacity(bucketCapacity int) Option {
	return func(l *TBLimiter) error {
		if bucketCapacity <= 0 {
			return fmt.Errorf("bucketCapacity must be greater than 0")
		}
		l.bucketCapacity = bucketCapacity
		return nil
	}
}

func WithClock(clock Clock) Option {
	return func(l *TBLimiter) error {
		if clock == nil {
			return fmt.Errorf("clock must not be nil ")
		}
		l.clock = clock
		return nil
	}
}

func WithObserveHooks(hooks ObserveHooks[string]) Option {
	return func(l *TBLimiter) error {
		l.hooks = hooks
		return nil
	}
}
