package limiter

import (
	"context"
)

type TokenBucketLimiter struct {
	rate  int
	store Store
}

func NewTokenBucket(rate int, store Store) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		rate:  rate,
		store: store,
	}
}

func (limiter *TokenBucketLimiter) Allow(ctx context.Context, key string) bool {
	return true
}

func (limiter *TokenBucketLimiter) Wait(ctx context.Context, key string) Error {
	return Error{}
}
