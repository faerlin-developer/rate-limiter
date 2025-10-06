package ratelimit

type DeniedError struct {
	Reason string
}

func NewDeniedError(reason string) *DeniedError {
	return &DeniedError{reason}
}

func (e DeniedError) Error() string {
	return e.Reason
}
