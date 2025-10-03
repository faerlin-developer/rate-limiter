# Rate Limiter

1. At least one fully working Rate Limiting Algorithms.
2. Per-key limiting with bounded memory: configurable max tracked keys +
   eviction policy (LRU or TTL or hybrid). Document your choice & trade-offs.
3. Concurrency safe.
4. Structured error for rate limit denials

```go
type Limiter interface {
    Allow(ctx context.Context, key string) bool
    Wait(ctx context.Context, key string) error // blocks until capacity or ctx done
}
```

```go
// Core interface required by the brief
type Limiter interface {
    Allow(ctx context.Context, key string) bool
    Wait(ctx context.Context, key string) error // blocks until capacity or ctx done
}

// Constructor surface (stable, ergonomic)
type Option func(*config)

func NewTokenBucket(opts ...Option) Limiter        // default, production-ready
func NewFixedWindow(opts ...Option) Limiter        // optional second algorithm
// (Internally both satisfy Limiter; callers pick what they need)
```

```go
lim := ratelimit.NewTokenBucket(
    ratelimit.WithRate(ratelimit.Rate{Events: 100, Per: time.Second}),
    ratelimit.WithBurst(10),                 // small burst after idle
    ratelimit.WithMaxKeys(100_000),          // memory cap
    ratelimit.WithEvictionTTL(10*time.Minute),
)

if !lim.Allow(ctx, userID) {
    // map to 429 + Retry-After based on DeniedError if using a helper
}

if err := lim.Wait(ctx, userID); err != nil {
    // ctx canceled/deadline; handle
}
```



## LRU
`github.com/hashicorp/golang-lru/v2`

