package ratelimit

import (
	"context"
	"github.com/faerlin-developer/rate-limiter.git/clock"
	"github.com/faerlin-developer/rate-limiter.git/db"
	"time"
)

type Bucket struct {
	lastRefillAt time.Time // time we last updated the number of tokens
	tokens       int       // tokens available as of lastUpdateAt
}

type TBLimiter struct {
	bucketCapacity int                  // maximum number of tokens at any time
	refillInterval time.Duration        // refill interval of token
	cache          TBCache              // key-value store for token-bucket algorithm
	clock          Clock                // Custom clock
	hooks          ObserveHooks[string] // Hooks for observability
}

func NewTBLimiter(options ...Option) (*TBLimiter, error) {

	cache, err := db.NewInMemoryCache[string, Bucket](DefaultCacheCapacity)
	if err != nil {
		return nil, err
	}

	// Default configuration of rate limiter
	limiter := &TBLimiter{
		bucketCapacity: DefaultBucketCapacity,
		refillInterval: time.Second / time.Duration(DefaultTokensPerSecond),
		clock:          clock.NewSystemClock(),
		cache:          cache,
		hooks:          NewEmptyObserveHooks[string](),
	}

	// Load user specified configuration
	for _, option := range options {
		err := option(limiter)
		if err != nil {
			return nil, err
		}
	}

	return limiter, nil
}

// Note the bootstrap behavior
func (l *TBLimiter) Allow(_ context.Context, key string) bool {

	// Create bucket if it does not exist; Get existing bucket otherwise.
	now := l.clock.Now()
	bucket, _ := l.cache.GetOrStore(key, l.freshBucket(now))

	// Lock current bucket
	l.cache.Lock(key)
	defer l.cache.Unlock(key)

	// Bucket may have updated while waiting to acquire lock; Get bucket again
	now = l.clock.Now()
	bucket, _ = l.cache.Get(key)

	l.refillBucket(&bucket, now)

	isAllowed := false
	if bucket.tokens > 0 {
		bucket.tokens--
		isAllowed = true
		l.hooks.OnAllow(key, now)
	} else {
		l.hooks.OnDeny(key, *NewDeniedError("insufficient token"))
	}

	l.cache.Put(key, bucket)

	return isAllowed
}

// Assume that a key is not evicted in the middle of calling Allow or Wait.
// Handled: Two goroutines calling Allow or Wait on the same Keys.

func (l *TBLimiter) Wait(ctx context.Context, key string) error {

	now := l.clock.Now()
	bucket, _ := l.cache.GetOrStore(key, l.freshBucket(now))

	for {
		l.cache.Lock(key)
		now := l.clock.Now()
		bucket, _ = l.cache.Get(key)

		l.refillBucket(&bucket, now)

		// Fast path
		if bucket.tokens > 0 {
			bucket.tokens--
			l.cache.Put(key, bucket)
			l.hooks.OnAllow(key, now)
			l.cache.Unlock(key)
			return nil
		}

		// Slow path
		elapsed := now.Sub(bucket.lastRefillAt)
		timeToWait := l.refillInterval - elapsed
		l.cache.Unlock(key)

		select {
		case <-ctx.Done():
			err := NewDeniedError(ctx.Err().Error())
			l.hooks.OnDeny(key, *err)
			return err
		case <-l.clock.After(timeToWait):
			continue
		}
	}
}

func (l *TBLimiter) freshBucket(now time.Time) Bucket {
	return Bucket{lastRefillAt: now, tokens: l.bucketCapacity}
}

func (l *TBLimiter) refillBucket(bucket *Bucket, now time.Time) {
	elapsed := now.Sub(bucket.lastRefillAt)
	tokensToAdd := int(elapsed / l.refillInterval)
	if elapsed > 0 && tokensToAdd > 0 {
		bucket.tokens = min(l.bucketCapacity, bucket.tokens+tokensToAdd)
		leftover := elapsed % l.refillInterval
		bucket.lastRefillAt = now.Add(-leftover)
	}
}
