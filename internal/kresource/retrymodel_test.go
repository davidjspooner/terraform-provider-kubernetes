package kresource

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func SameErrorMessages(err1, err2 error) bool {
	var msg1, msg2 string
	if err1 != nil {
		msg1 = err1.Error()
	}
	if err2 != nil {
		msg2 = err2.Error()
	}
	return msg1 == msg2
}

func PtrTo[T any](v T) *T {
	return &v
}

func TestRetryHelper(t *testing.T) {
	retrySchema := RetryModel{
		MaxAttempts: PtrTo[int64](3),
		FastFail:    PtrTo([]string{"abort"}),
		Pause:       PtrTo("2s"),
		Interval:    PtrTo("1s,2s,3s"),
		Timeout:     PtrTo("10s"),
	}

	retryHelper, err := retrySchema.NewHelper(nil)
	if err != nil {
		t.Fatalf("RetrySchema.NewHelper() = [%v], expected [nil]", err)
	}

	ctx := context.Background()

	testCases := []struct {
		fn            func(ctx context.Context, attempt int) error
		expectedError error
	}{
		{
			fn: func(ctx context.Context, attempt int) error {
				return nil
			},
			expectedError: fmt.Errorf(""),
		},
		{
			fn: func(ctx context.Context, attempt int) error {
				return fmt.Errorf("error occurred")
			},
			expectedError: fmt.Errorf("aborted after 3 attempt(s), error occurred"),
		},
		{
			fn: func(ctx context.Context, attempt int) error {
				if attempt < 2 {
					return fmt.Errorf("success on 2nd attempt occurred")
				}
				return nil
			},
			expectedError: fmt.Errorf(""),
		},
		{
			fn: func(ctx context.Context, attempt int) error {
				if attempt < 5 {
					return fmt.Errorf("success on 5th attempt occurred")
				}
				return nil
			},
			expectedError: fmt.Errorf("aborted after 3 attempt(s), success on 5th attempt occurred"),
		},
		{
			fn: func(ctx context.Context, attempt int) error {
				time.Sleep(11 * time.Second)
				return fmt.Errorf("slow error occurred")
			},
			expectedError: fmt.Errorf("context deadline exceeded, slow error occurred"),
		},
		{
			fn: func(ctx context.Context, attempt int) error {
				return fmt.Errorf("special abort state")
			},
			expectedError: fmt.Errorf("fast fail, special abort state"),
		},
	}
	for testNumber, tc := range testCases {
		err := retryHelper.Retry(ctx, tc.fn)
		if err != nil {
			t.Fatal(err.Error())
		}
		ctx, cancel := retryHelper.SetDeadline(ctx)
		_ = ctx
		defer cancel()
		if !SameErrorMessages(err, tc.expectedError) {
			t.Errorf("Test #%d : RetryHelper.Retry() = [%v], expected [%s]", testNumber+1, err, tc.expectedError)
		}
	}
}