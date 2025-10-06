package clock

import (
	"sync"
	"time"
)

type TestClock struct {
	mutex   sync.Mutex
	now     time.Time
	waiters []waiter
}

type waiter struct {
	at time.Time
	ch chan time.Time
}

func NewTestClock(start time.Time) *TestClock {
	return &TestClock{now: start}
}

func (c *TestClock) Now() time.Time {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.now
}

func (c *TestClock) After(d time.Duration) <-chan time.Time {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	ch := make(chan time.Time, 1) // buffered so delivery can't block
	fireAt := c.now.Add(d)

	if d <= 0 {
		// deliver immediately based on current logical time
		ch <- fireAt
		return ch
	}

	c.waiters = append(c.waiters, waiter{at: fireAt, ch: ch})
	return ch
}

func (c *TestClock) Forward(d time.Duration) {
	// advance logical time and collect due waiters
	c.mutex.Lock()
	c.now = c.now.Add(d)
	now := c.now

	// collect and remove all due waiters without holding the lock while sending
	var due []waiter
	for i := 0; i < len(c.waiters); {
		if !c.waiters[i].at.After(now) { // at <= now
			due = append(due, c.waiters[i])
			// remove by swapping with last
			last := len(c.waiters) - 1
			c.waiters[i] = c.waiters[last]
			c.waiters = c.waiters[:last]
			continue
		}
		i++
	}
	c.mutex.Unlock()

	// deliver outside the lock
	for _, w := range due {
		select {
		case w.ch <- w.at:
		default:
			// shouldn't happen (buffered 1), but avoid blocking tests
		}
	}
}
