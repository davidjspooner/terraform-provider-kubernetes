package job

import (
	"context"
	"fmt"
	"regexp"
	"time"
)

type RetryHelper struct {
	MaxAttempts int
	FastFail    []*regexp.Regexp
	Pause       time.Duration
	Interval    DurationList
	Timeout     time.Duration
}

func (rh *RetryHelper) SetDeadline(ctx context.Context) (context.Context, context.CancelFunc) {
	rh.SetDefaults()
	return context.WithDeadline(ctx, time.Now().Add(rh.Timeout))
}

func (rh *RetryHelper) SetDefaults() {
	if rh.Interval == nil {
		rh.Interval = []time.Duration{10 * time.Second, 20 * time.Second, 30 * time.Second}
	}
	if rh.MaxAttempts == 0 {
		rh.MaxAttempts = 1000 * 1000 * 1000 // 1 billion time should be enough
	}
	if rh.Timeout == 0 {
		rh.Timeout = 5 * time.Minute
	}
}

func (rh *RetryHelper) Retry(ctx context.Context, fn func(ctx context.Context, attempt int) error) error {

	//tflog.Info(ctx, "waiting", map[string]any{"duration": d.String()}) //TODO add some form of dependancy logging
	Sleep(ctx, rh.Pause)

	err := &RetryError{}

	for {
		if err.Attempt >= rh.MaxAttempts {
			err.AbortReason = fmt.Errorf("aborted after %d attempt(s)", err.Attempt)
			return err
		}
		err.AbortReason = ctx.Err()
		if err.AbortReason != nil {
			return err
		}
		if err.Attempt > 0 {
			interval := rh.Interval[Min(err.Attempt-1, len(rh.Interval)-1)]
			//tflog.Info(ctx, "waiting", map[string]any{"duration": d.String()}) //TODO add some form of dependancy logging
			Sleep(ctx, interval)
			err.AbortReason = ctx.Err()
			if err.AbortReason != nil {
				return err
			}
		}
		err.Attempt++
		err.RecentAttempt = fn(ctx, err.Attempt)
		if err.RecentAttempt == nil {
			return nil
		}
		currentErrStr := err.RecentAttempt.Error()
		for _, hint := range rh.FastFail {
			if hint.FindStringIndex(currentErrStr) != nil {
				err.AbortReason = fmt.Errorf("fast fail")
				return err
			}
		}
	}
}
