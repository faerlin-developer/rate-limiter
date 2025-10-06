package ratelimit

import (
	"context"
)

type Limiter interface {

	// Allow performs a non-blocking check to determine if the key is permitted.
	// If permitted, return true; otherwise, return false.
	Allow(ctx context.Context, key string) bool

	// Wait blocks until the key is permitted or context is done.
	// If the context is done, return a DeniedError; otherwise, return nil.
	Wait(ctx context.Context, key string) error
}
