package ratelimit

import (
	"context"
	"github.com/faerlin-developer/rate-limiter.git/clock"
	"github.com/faerlin-developer/rate-limiter.git/db"
	"time"
)

// Bucket is the value stored in the cache for a given key
type Bucket struct {
	lastRefillAt time.Time // last time the number of tokens was updated
	tokens       int       // tokens available as of lastRefillAt
}

// TBLimiter Rate limiter with the token bucket algorithm.
type TBLimiter struct {
	bucketCapacity int                  // Maximum number of tokens at any time
	refillInterval time.Duration        // Refill interval of token
	cache          TBCache              // Key-value store for token-bucket algorithm
	clock          Clock                // Custom clock
	hooks          ObserveHooks[string] // Hooks for observability
}

// NewTBLimiter returns a rate limiter that uses the token bucket algorithm.
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

// Allow performs a non-blocking check to determine if the key is permitted.
// If permitted, return true; otherwise, return false.
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

// Wait blocks until the key is permitted or context is done.
// If the context is done, return a DeniedError; otherwise, return nil.
func (l *TBLimiter) Wait(ctx context.Context, key string) error {

	now := l.clock.Now()
	bucket, _ := l.cache.GetOrStore(key, l.freshBucket(now))

	for {

		// Acquire the per-key lock
		l.cache.Lock(key)

		// The bucket may have been updated while waiting to acquire the lock; Get the latest bucket
		bucket, _ = l.cache.Get(key)
		now := l.clock.Now()

		// Refill the bucket
		l.refillBucket(&bucket, now)

		// Fast path: consume a token and return
		if bucket.tokens > 0 {
			bucket.tokens--
			l.cache.Put(key, bucket)
			l.hooks.OnAllow(key, now)
			l.cache.Unlock(key)
			return nil
		}

		// Slow path: compute timeToWait and then block until wake-up time.
		elapsed := now.Sub(bucket.lastRefillAt)
		timeToWait := l.refillInterval - elapsed

		// Release the lock before blocking to wait for wake-up time.
		l.cache.Unlock(key)

		// Wait for wake-up time or context done, whichever comes first.
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

// freshBucket returns a bucket with a capacity specified by bucketCapacity.
func (l *TBLimiter) freshBucket(now time.Time) Bucket {
	return Bucket{lastRefillAt: now, tokens: l.bucketCapacity}
}

// refillBucket adds tokens to given bucket according to the elapsed time since the last refill.
func (l *TBLimiter) refillBucket(bucket *Bucket, now time.Time) {
	elapsed := now.Sub(bucket.lastRefillAt)
	tokensToAdd := int(elapsed / l.refillInterval)
	if elapsed > 0 && tokensToAdd > 0 {
		bucket.tokens = min(l.bucketCapacity, bucket.tokens+tokensToAdd)
		leftover := elapsed % l.refillInterval
		bucket.lastRefillAt = now.Add(-leftover)
	}
}
