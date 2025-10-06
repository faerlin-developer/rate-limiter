# Rate Limiter

This is an initial implementation of a rate limiter module in Go. 

## Key Features

__Algorithm__: Implements the [Token Bucket](https://www.hellointerview.com/learn/system-design/problem-breakdowns/distributed-rate-limiter#:~:text=to%20implement%20correctly.-,Token%20Bucket,-Think%20of%20each) algorithm. Each key is associated with a bucket that holds upto `bucketCapacity` tokens. Tokens are added at a steady rate of `tokensPerSecond`. A request is allowed by spending one token. The request is denied when there are no tokens to spend.

__Concurrency__: The rate limiter is safe for concurrent use, including overlapping calls to `Allow` and `Wait` on the same key as well as on different keys. Thread-safe on overlapping calls to `Allow` and `Wait` on the same key is made possible by per-key mutexes. The `InMemoryCache` is also thread safe as access to the underlying LRU cache is protected by a single mutex, which makes overlapping calls to `Allow` and `Wait` on different keys safe for concurrent use. 

__Observability__: Hooks for observability `OnAllow` and `OnDeny` can be defined as options when creating the rate limiter.

__Ergonomics__: The rate limiter is parameterized by a `Cache` interface, so any cache (LRU, TTL, in-memory, Redis-backed) can be substituted without changing limiter code.

__Structured Error__: For rate denials, `Wait` returns an instance of a user-defined error `DeniedError`. 

__Cache__: The rate limiter keeps per-key state in a cache to make quick allow/deny decisions. The rate limiter uses a cache that implements the `Cache` interface in `db/cache.go`. An in-memory cache implementation called `InMemoryCache` is provided in `db/in_memory_cache.go`. This implementation is backed by an LRU cache provided by `hashicorp/golang-lru`. The following eviction policies were considered:

| Eviction Policy | Pros                                                              | Cons                                                             | Best for workloads with          |
|-----------------|-------------------------------------------------------------------|------------------------------------------------------------------|----------------------------------|
| LRU             | Keeps frequently used keys in cache                               | Sudden burst of many one-off keys can evict frequently used keys | stable hot keys                  |
| TTL             | Cleans up one-off keys                                            | On its own, it has no peak protection                            | pre-dominantly one-off keys      |
| Hybrid          | Combines peak protection with background clean up of one-off keys | More bookkeeping (use doubly linked list and heap)               | stable hot keys and one-off keys |

## Assumption
We currently assume that a key is not evicted from the cache during an in-flight `Allow` or `Wait` on that key. Eliminating this assumption requires more time and is deferred to future work.

## Quick Start

```go
// Options
tokensPerSecond := 1
bucketCapacity := 1
hooks := ratelimit.NewEmptyObserveHooks[string]()
cache, _ := db.NewInMemoryCache[string, ratelimit.Bucket](100)

// Create Rate Limiter (Token Bucket Algorithm)
limiter, _ := ratelimit.NewTBLimiter(
	ratelimit.WithTokensPerSecond(tokensPerSecond),
	ratelimit.WithBucketCapacity(bucketCapacity),
	ratelimit.WithCache(cache),
	ratelimit.WithObserveHooks(hooks))

// Call Allow(k)
pass := limiter.Allow(context.Background(), "74.125.200.113")
if !pass {
	fmt.Printf("Allow(%s) denied\n", key)
} 

// Call Wait(k)
err = limiter.Wait(context.Background(), "74.125.200.113")
if err != nil {
	fmt.Printf("Wait(%s) denied: %s\n", key, err.Error())
} 
```

## Demo

Run the demo in `demo/main.go` with:

```bash
make demo
```

## Unit Tests

Run the unit tests in `ratelimit/token_bucket_test.go` with:

```bash 
make test
```

<!-- 
1. Allow and Wait
2. At least one fully working Rate Limiting Algorithms.
2. Per-key limiting with bounded memory: configurable max tracked keys +
   eviction policy (LRU or TTL or hybrid). Document your choice & trade-offs.
3. Concurrency safe.
4. Structured error for rate limit denials
-->

## References

1. [golang-lru by hashicorp](github.com/hashicorp/golang-lru/v2)
2. [Design a Rate Limiter by HelloInterview](https://www.hellointerview.com/learn/system-design/problem-breakdowns/distributed-rate-limiter)
