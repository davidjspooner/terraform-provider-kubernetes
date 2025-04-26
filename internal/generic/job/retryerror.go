package job

import "fmt"

type RetryError struct {
	RecentAttempt error
	AbortReason   error
	Attempt       int
}

func (e *RetryError) Error() string {
	return fmt.Sprintf("%v, %v", e.AbortReason, e.RecentAttempt)
}
