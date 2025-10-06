package ratelimit

import (
	"context"
	"errors"
	"github.com/faerlin-developer/rate-limiter.git/clock"
	"sync"
	"testing"
	"time"
)

func Test_Allow_ConsumesAndRefills(t *testing.T) {

	limiter, testClock := newTestFixture(t, 1, 1)
	ctx := context.Background()

	// First Allow(k) should pass
	if !limiter.Allow(ctx, "k") {
		t.Fatal("first Allow(k) = false; want true")
	}

	// Second Allow(k) should fail
	if limiter.Allow(ctx, "k") {
		t.Fatal("second Allow(k) = true; want false (no tokens left)")
	}

	// Forward the clock by 1s to replenish one token
	testClock.Forward(1 * time.Second)

	// Third Allow(k) should pass
	if !limiter.Allow(ctx, "k") {
		t.Fatal("after 1s, Allow(k) = false; want true")
	}
}

func Test_Allow_PerKeyIsolation(t *testing.T) {

	limiter, _ := newTestFixture(t, 1, 1)
	ctx := context.Background()

	// Allow(A) should pass
	if !limiter.Allow(ctx, "A") {
		t.Fatal("Allow(A) = false; want true")
	}

	// Bucket for key A is empty, but Allow(B) should pass
	if !limiter.Allow(ctx, "B") {
		t.Fatal("Allow(B) = false; want true (per-key isolation)")
	}
}

func Test_Allow_ConcurrentSameKey(t *testing.T) {

	limiter, _ := newTestFixture(t, 1, 1)
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(2)

	results := make([]bool, 2)
	for i := 0; i < 2; i++ {
		go func(index int) {
			defer wg.Done()
			results[index] = limiter.Allow(ctx, "k")
		}(i)
	}

	wg.Wait()

	// Exactly one true, one false.
	trueCount := 0
	for _, r := range results {
		if r {
			trueCount++
		}
	}
	
	if trueCount != 1 {
		t.Fatalf("Allow concurrency: got %v, want exactly one true", results)
	}
}

func Test_Wait_BlocksUntilRefill(t *testing.T) {

	// 2 tokens per second
	limiter, testClock := newTestFixture(t, 2, 1)
	ctx := context.Background()

	// Drain the single token.
	if !limiter.Allow(ctx, "k") {
		t.Fatal("precondition failed; expected first Allow to succeed")
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := limiter.Wait(ctx, "k"); err != nil {
			t.Errorf("Wait returned error: %v", err)
		}
	}()

	// Show that Wait(k) is still blocking
	select {
	case <-done:
		t.Fatal("Wait returned before refill; should be blocked")
	case <-time.After(10 * time.Millisecond):
	}

	// Forward the block by 500 ms to replenish one token
	testClock.Forward(500 * time.Millisecond)

	select {
	case <-done:
	case <-time.After(50 * time.Millisecond):
		t.Fatal("Wait did not return after token became available")
	}
}

func Test_Wait_ContextCancel(t *testing.T) {

	limiter, _ := newTestFixture(t, 1, 1)

	// Drain the single token.
	if !limiter.Allow(context.Background(), "k") {
		t.Fatal("precondition failed; expected first Allow to succeed")
	}

	ctxWithCancel, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err := limiter.Wait(ctxWithCancel, "k")
	if err == nil {
		t.Fatal("Wait returned nil; want context error")
	}

	var e *DeniedError
	if !errors.As(err, &e) {
		t.Fatal("Wait return unexpected error type; want DeniedError")
	}
}

func Test_Wait_ConcurrentSameKey_OnlyOneProceedsImmediately(t *testing.T) {

	limiter, testClock := newTestFixture(t, 1, 1)

	// one token available now
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	results := make([]error, 2)
	results[0] = NewDeniedError("denied")
	results[1] = NewDeniedError("denied")

	for i := 0; i < 2; i++ {
		go func(index int) {
			defer wg.Done()
			<-start
			results[index] = limiter.Wait(context.Background(), "k")
		}(i)
	}

	close(start)

	// Let the first waiter take the immediate token; the second should block until we advance time.
	time.Sleep(50 * time.Millisecond) // scheduling only; not for timing logic

	// No time advanced yet: exactly one should be done.
	if (results[0] == nil && results[1] == nil) || (results[0] != nil && results[1] != nil) {
		t.Fatalf("after start: done not 1, want 1")
	}

	testClock.Forward(1 * time.Second) // refill another token
	wg.Wait()                          // wait for the

	// Both should succeed
	for i, err := range results {
		if err != nil {
			t.Fatalf("Waiter %d error = %v; want nil", i, err)
		}
	}
}

func newTestClock() *clock.TestClock {
	startTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	return clock.NewTestClock(startTime)
}

func newTestFixture(t *testing.T, tokensPerSecond int, bucketCapacity int) (*TBLimiter, *clock.TestClock) {

	testClock := newTestClock()
	limiter, err := NewTBLimiter(
		WithTokensPerSecond(tokensPerSecond),
		WithBucketCapacity(bucketCapacity),
		WithClock(testClock))

	if err != nil {
		t.Fatal("failed to create token bucket limiter")
	}

	return limiter, testClock
}
