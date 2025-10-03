package limiter

import (
	"fmt"
	"time"
)

type Error struct {
	Key        string
	RetryAfter time.Duration // Retry after
	Algorithm  string        // "token_bucket", "fixed_window"
}

func (e *Error) Error() string {
	return fmt.Sprintf("rate limited with key=%s retry_after=%s algorithm=%s", e.Key, e.RetryAfter, e.Algorithm)
}
