package ratelimit

import (
	"context"
	"fmt"
	"github.com/faerlin-developer/rate-limiter.git/clock"
	"github.com/faerlin-developer/rate-limiter.git/db"
	"time"
)

type Clock clock.Clock
type TBCache db.Cache[string, Bucket]
type Option func(limiter *TBLimiter)

func WithCache(cache TBCache) Option {
	return func(l *TBLimiter) { l.cache = cache }
}

func WithClock(clock Clock) Option {
	return func(l *TBLimiter) { l.clock = clock }
}

type TBLimiter struct {
	bucketCapacity int           // maximum number of tokens at any time
	refillInterval time.Duration // refill interval of token
	cache          TBCache       // key-value store for token-bucket algorithm
	clock          Clock
}

type Bucket struct {
	lastRefillAt time.Time // time we last updated the number of tokens
	tokens       int       // tokens available as of lastUpdateAt
}

func NewTBLimiter(requestsPerSecond int, options ...Option) (*TBLimiter, error) {

	if requestsPerSecond <= 0 {
		return nil, fmt.Errorf("requests per second must be greater than 0")
	}

	cache, err := db.NewInMemoryCache[string, Bucket](100)
	if err != nil {
		return nil, err
	}

	limiter := &TBLimiter{
		bucketCapacity: requestsPerSecond,
		refillInterval: time.Second / time.Duration(requestsPerSecond),
		cache:          cache,
	}

	return limiter, nil
}

// Note the bootstrap behavior
func (l *TBLimiter) Allow(ctx context.Context, key string) bool {

	now := time.Now()
	bucket, _ := l.cache.GetOrStore(key, l.freshBucket(now))

	l.cache.Lock(key)
	defer l.cache.Unlock(key)

	l.refillBucket(&bucket, now)
	isAllowed := false
	if bucket.tokens > 0 {
		bucket.tokens--
		isAllowed = true
	}

	l.cache.Put(key, bucket)

	return isAllowed
}

// Assume that a key is not evicted in the middle of calling Allow or Wait.
// Handled: Two goroutines calling Allow or Wait on the same Keys.

func (l *TBLimiter) Wait(ctx context.Context, key string) error {

	now := time.Now()
	bucket, _ := l.cache.GetOrStore(key, l.freshBucket(now))

	for {
		l.cache.Lock(key)
		now := time.Now()
		bucket, _ = l.cache.Get(key)

		l.refillBucket(&bucket, now)

		// Fast path
		if bucket.tokens > 0 {
			bucket.tokens--
			l.cache.Put(key, bucket)
			l.cache.Unlock(key)
			return nil
		}

		// Slow path
		elapsed := now.Sub(bucket.lastRefillAt)
		timeToWait := l.refillInterval - elapsed
		l.cache.Unlock(key)

		select {
		case <-ctx.Done():
			return fmt.Errorf("context done")
		case <-time.After(timeToWait):
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
