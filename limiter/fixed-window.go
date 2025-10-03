package limiter

type FixedWindowLimiter struct{}

func NewFixedWindow(rate int, store Store) *TokenBucketLimiter {
	return nil
}
