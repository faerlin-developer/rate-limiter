package ratelimit

import (
	"context"
	"errors"
	"github.com/faerlin-developer/rate-limiter.git/clock"
	"sync"
	"testing"
	"time"
)

func Test_Allow_SameKey_ConsumesAndRefillsToken(t *testing.T) {

	// Create rate limiter with 1 token/second and bucket capacity of 1
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

func Test_Allow_DifferentKeys_ConsumeTokens(t *testing.T) {

	// Create rate limiter with 1 token/second and bucket capacity of 1
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

func Test_Allow_ConcurrentSameKey_ConsumeExactlyOneToken(t *testing.T) {

	// Create rate limiter with 1 token/second and bucket capacity of 1
	limiter, _ := newTestFixture(t, 1, 1)
	ctx := context.Background()

	// Wait group for two goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	// Will contain the output of calling Allow by two goroutines
	results := make([]bool, 2)

	// Launch two goroutines that calls Allow on the same key
	for i := 0; i < 2; i++ {
		go func(index int) {
			defer wg.Done()
			results[index] = limiter.Allow(ctx, "k")
		}(i)
	}

	wg.Wait()

	// Count the number of pass issued by Allow
	passCount := 0
	for _, isAllowed := range results {
		if isAllowed {
			passCount++
		}
	}

	// Expect only one pass from the two calls on Allow on the same key
	if passCount != 1 {
		t.Fatalf("Allow concurrency: got %v, want exactly one true", results)
	}
}

func Test_Wait_SameKey_BlocksUntilRefill(t *testing.T) {

	// Create rate limiter with 2 token/second and bucket capacity of 1
	limiter, testClock := newTestFixture(t, 2, 1)
	ctx := context.Background()

	// Drain the single token from the bucket by calling Allow(k)
	if !limiter.Allow(ctx, "k") {
		t.Fatal("precondition failed; expected first Allow to succeed")
	}

	// Launch goroutine that calls Wait(k)
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
	case <-time.After(100 * time.Millisecond):
		break
	}

	// Forward the block by 500 ms to replenish one token
	testClock.Forward(500 * time.Millisecond)

	// Show that Wait(k) is no longer blocking
	select {
	case <-done:
		break
	case <-time.After(50 * time.Millisecond):
		t.Fatal("Wait did not return after token became available")
	}
}

func Test_Wait_ContextCancel(t *testing.T) {

	// Create rate limiter with 2 token/second and bucket capacity of 1
	limiter, _ := newTestFixture(t, 1, 1)

	// Drain the single token from the bucket by calling Allow(k)
	if !limiter.Allow(context.Background(), "k") {
		t.Fatal("precondition failed; expected first Allow to succeed")
	}

	// Create a context with timeout
	ctxWithCancel, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	// Call Wait(k) without advancing the clock
	err := limiter.Wait(ctxWithCancel, "k")
	if err == nil {
		t.Fatal("Wait returned nil; want context error")
	}

	// Expect deny due to timeout
	var e *DeniedError
	if !errors.As(err, &e) {
		t.Fatal("Wait return unexpected error type; want DeniedError")
	}
}

func Test_Wait_ConcurrentSameKey_OnlyOneProceedsImmediately(t *testing.T) {

	// Create rate limiter with 1 token/second and bucket capacity of 1
	limiter, testClock := newTestFixture(t, 1, 1)

	numThreads := 2
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(numThreads)

	// Initialize result array with non-nil errors
	results := make([]error, numThreads)
	for i, _ := range results {
		results[i] = NewDeniedError("")
	}

	// Launch goroutines that call Wait(k) on the same key
	for i := 0; i < numThreads; i++ {
		go func(index int) {
			defer wg.Done()
			<-start
			results[index] = limiter.Wait(context.Background(), "k")
		}(i)
	}

	close(start)

	// Let the first waiter take the immediate token; the second should block until we advance time.
	time.Sleep(50 * time.Millisecond)

	// At this point, we expect only one call on Wait(k) to return and pass
	passCount := 0
	for i, _ := range results {
		if results[i] == nil {
			passCount++
		}
	}

	if passCount != 1 {
		t.Fatalf("after start: done not 1, want 1")
	}

	// Forward the time to refill another token
	testClock.Forward(1 * time.Second)
	wg.Wait()

	// Both calls on Wait(k) should have succeed
	for i, err := range results {
		if err != nil {
			t.Fatalf("Waiter %d error = %v; want nil", i, err)
		}
	}
}

// newTestClock returns an instance of the TestClock with a starting time at 2000-01-01.
func newTestClock() *clock.TestClock {
	startTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	return clock.NewTestClock(startTime)
}

// newTestFixture returns a rate limiter for testing.
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
