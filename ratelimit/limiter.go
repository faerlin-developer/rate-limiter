package ratelimit

import (
	"context"
)

// context.canceled, context.DeadlineExceeded
// context.Done
// safe for concurrent use
type Limiter interface {
	Allow(ctx context.Context, key string) bool
	Wait(ctx context.Context, key string) error
}
