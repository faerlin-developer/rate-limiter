package clock

import (
	"sync"
	"time"
)

// TestClock is the Clock used for testing
type TestClock struct {
	mutex   sync.Mutex
	now     time.Time
	waiters []waiter
}

// waiter represents a goroutine waiting until a specified future time.
type waiter struct {
	wakeAt time.Time
	notify chan time.Time
}

// NewTestClock returns an instance of a test clock with the specified start time.
func NewTestClock(start time.Time) *TestClock {
	return &TestClock{now: start}
}

// Now returns the time now
func (c *TestClock) Now() time.Time {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.now
}

// After Returns a channel that returns the time after the specified duration
// Important: To avoid blocking in Forward, the caller must begin receiving
// from the returned channel immediately.
func (c *TestClock) After(d time.Duration) <-chan time.Time {

	c.mutex.Lock()
	defer c.mutex.Unlock()

	// The future time the goroutine is waiting for
	wakeAt := c.now.Add(d)

	// Use a buffered channel so that we don't block on send
	notifyChannel := make(chan time.Time, 1)

	// Check if we can notify the channel immediately
	if d <= 0 {
		notifyChannel <- wakeAt
		return notifyChannel
	}

	// Add as one of the waiters
	// When the wakeAt time of the waiter is reached, notify the channel.
	c.waiters = append(c.waiters, waiter{wakeAt: wakeAt, notify: notifyChannel})

	// Return channel for the goroutine to block on
	return notifyChannel
}

// Forward advances the Clock's time forward by the specified duration.
func (c *TestClock) Forward(d time.Duration) {

	c.mutex.Lock()
	c.now = c.now.Add(d)
	now := c.now

	// Collect waiters whose wake up times are due
	var due []waiter
	for i := 0; i < len(c.waiters); {
		if c.isWaiterDue(i, now) {
			due = append(due, c.waiters[i])
			c.removeWaiter(i)
			continue
		}
		i++
	}

	// We can release the lock at this point
	c.mutex.Unlock()

	// Notify the channels of due waiters
	for _, w := range due {
		w.notify <- w.wakeAt
	}
}

// removeWaiter removes the specified waiter from the list of waiters.
func (c *TestClock) removeWaiter(index int) {

	// Remove by swapping with last
	last := len(c.waiters) - 1
	c.waiters[index] = c.waiters[last]
	c.waiters = c.waiters[:last]
}

// isWaiterDue returns true once the waiter has reached its wake up time.
func (c *TestClock) isWaiterDue(index int, now time.Time) bool {
	return !c.waiters[index].wakeAt.After(now)
}
