package main

import (
	"context"
	"fmt"
	"github.com/faerlin-developer/rate-limiter.git/db"
	"github.com/faerlin-developer/rate-limiter.git/ratelimit"
)

func main() {

	// Options
	tokensPerSecond := 1
	bucketCapacity := 1
	hooks := ratelimit.NewEmptyObserveHooks[string]()
	cache, err := db.NewInMemoryCache[string, ratelimit.Bucket](100)
	if err != nil {
		panic(err)
	}

	// Create Rate Limiter (Token Bucket Algorithm)
	limiter, err := ratelimit.NewTBLimiter(
		ratelimit.WithTokensPerSecond(tokensPerSecond),
		ratelimit.WithBucketCapacity(bucketCapacity),
		ratelimit.WithCache(cache),
		ratelimit.WithObserveHooks(hooks))

	if err != nil {
		panic("failed to create rate limiter")
	}

	ctx := context.Background()
	key := "74.125.200.113"

	// Call Allow(k)
	pass := limiter.Allow(ctx, key)
	if pass {
		fmt.Printf("Allow(%s) passed\n", key)
	} else {
		fmt.Printf("Allow(%s) denied\n", key)
	}

	// Call Wait(k)
	err = limiter.Wait(ctx, key)
	if err == nil {
		fmt.Printf("Wait(%s) passed\n", key)
	} else {
		fmt.Printf("Wait(%s) denied: %s\n", key, err.Error())
	}
}
