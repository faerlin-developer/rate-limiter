package ratelimit

import (
	"fmt"
)

type Error struct {
	Reason string
}

func (e *Error) Error() string {
	return fmt.Sprintf("failed")
}
