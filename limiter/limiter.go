package limiter

import (
	"context"
	"github.com/faerlin-developer/rate-limiter.git/store"
)

type Store = store.Store

// context.canceled, context.DeadlineExceeded
// context.Done
// safe for concurrent use
type Limiter interface {
	Allow(ctx context.Context, key string) bool
	Wait(ctx context.Context, key string) Error
}
