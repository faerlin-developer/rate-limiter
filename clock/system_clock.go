package clock

import "time"

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
