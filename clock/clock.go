package clock

import "time"

type Clock interface {
	Now() time.Time
	After(duration time.Duration) <-chan time.Time
}
