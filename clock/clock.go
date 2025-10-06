package clock

import "time"

// Clock is an interface for keeping track of time
type Clock interface {

	// Now returns the time now
	Now() time.Time

	// After Returns a channel that returns the time after the specified duration
	After(duration time.Duration) <-chan time.Time
}
