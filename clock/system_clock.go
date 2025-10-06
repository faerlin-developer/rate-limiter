package clock

import "time"

// SystemClock provides the default Clock for the rate limiter under normal use.
type SystemClock struct{}

func NewSystemClock() *SystemClock {
	return &SystemClock{}
}

func (c *SystemClock) Now() time.Time {
	return time.Now()
}

func (c *SystemClock) After(duration time.Duration) <-chan time.Time {
	return time.After(duration)
}
