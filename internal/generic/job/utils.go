package job

import (
	"context"
	"time"

	"golang.org/x/exp/constraints"
)

func Sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func Min[T constraints.Ordered](a T, others ...T) T {
	least := a
	for _, v := range others {
		if v < least {
			least = v
		}
	}
	return least
}
func Max[T constraints.Ordered](a T, others ...T) T {
	least := a
	for _, v := range others {
		if v < least {
			least = v
		}
	}
	return least
}

type Logger interface {
	Printf(format string, v ...interface{})
}

type LoggerFunc func(format string, v ...interface{})

func (f LoggerFunc) Printf(format string, v ...interface{}) {
	f(format, v...)
}
