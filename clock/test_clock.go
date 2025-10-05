package clock

import (
	"sync"
	"time"
)

type FakeClock struct {
	mutex sync.Mutex
	now   time.Time
}

func NewFakeClock(start time.Time) *FakeClock {
	return &FakeClock{now: start}
}

func (c *FakeClock) Now() time.Time {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.now
}

func (c *FakeClock) After(d time.Duration) <-chan time.Time {

	c.now.Add(d)
	ch := make(chan time.Time)
	go func() {
		ch <- c.now
	}()

	return ch
}

func (c *FakeClock) Forward(d time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.now = c.now.Add(d)
}
