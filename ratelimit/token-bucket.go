package ratelimit

import (
	"context"
	"fmt"
	"github.com/faerlin-developer/rate-limiter.git/db"
	"sync"
	"time"
)

type TBLimiter struct {
	bucketCapacity int           // maximum number of tokens at any time
	refillInterval time.Duration // refill interval of token
	store          TBStore       // key-value store for token-bucket algorithm
	muByKey        sync.Map      // Maps a key to a mutex
}

type TBStore = db.Store[string, Bucket]

type Bucket struct {
	lastRefillAt time.Time // time we last updated the number of tokens
	tokens       int       // tokens available as of lastUpdateAt
}

// Note burst capacity
func NewTBLimiter(requestsPerSecond int, store TBStore) (*TBLimiter, error) {

	if requestsPerSecond <= 0 {
		return nil, fmt.Errorf("requests per second and bucket capacity must be greater than 0")
	}

	//
	bucketCapacity := requestsPerSecond

	//
	refillInterval := time.Second / time.Duration(requestsPerSecond)

	return &TBLimiter{
		bucketCapacity: bucketCapacity,
		refillInterval: refillInterval,
		store:          store,
	}, nil
}

// Needs a lock for each key
// Add the lock in Bucket value

// Note the bootstrap behavior
func (l *TBLimiter) Allow(ctx context.Context, key string) bool {

	now := time.Now()
	bucket, ok := l.store.Get(key)

	if !ok {
		bucket = Bucket{lastRefillAt: now, tokens: l.bucketCapacity}
	} else {
		l.refill(&bucket, now)
	}

	isAllowed := false
	if bucket.tokens > 0 {
		bucket.tokens--
		isAllowed = true
	}

	l.store.Put(key, bucket)

	return isAllowed
}

func (l *TBLimiter) refill(bucket *Bucket, now time.Time) {
	elapsed := now.Sub(bucket.lastRefillAt)
	tokensToAdd := int(elapsed / l.refillInterval)
	if elapsed > 0 && tokensToAdd > 0 {
		bucket.tokens = min(l.bucketCapacity, bucket.tokens+tokensToAdd)
		leftover := elapsed % l.refillInterval
		bucket.lastRefillAt = now.Add(-leftover)
	}
}

func (l *TBLimiter) Wait(ctx context.Context, key string) error {

	now := time.Now()
	bucket, ok := l.store.Get(key)

	var err error
	if !ok {
		bucket = Bucket{lastRefillAt: now, tokens: l.bucketCapacity - 1}
	} else {

		l.refill(&bucket, now)

		if bucket.tokens == 0 {

			elapsed := now.Sub(bucket.lastRefillAt)
			timeToWait := l.refillInterval - elapsed

			if timeToWait >= 0 {
				select {
				case <-ctx.Done():
					err = fmt.Errorf("context done")
				case <-time.After(timeToWait):
					err = nil
				}
			}

			bucket.lastRefillAt = time.Now()
		} else {
			bucket.tokens--
		}
	}

	l.store.Put(key, bucket)

	return err
}

// lockFor returns the *same* mutex for the same key every time.
func (l *TBLimiter) lockFor(key string) *sync.Mutex {
	// Fast path: does a concurrent read; if present, reuse it.
	if v, ok := l.muByKey.Load(key); ok {
		return v.(*sync.Mutex)
	}

	// Slow path: allocate a new mutex candidate.
	m := &sync.Mutex{}

	// Atomically publish the mutex if key absent; otherwise reuse existing.
	actual, loaded := l.muByKey.LoadOrStore(key, m)
	if loaded {
		// Someone else stored one first; discard ours and use theirs.
		return actual.(*sync.Mutex)
	}
	// We successfully installed our new mutex.
	return m
}
