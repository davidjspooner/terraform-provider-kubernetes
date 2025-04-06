package job

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestParseList(t *testing.T) {
	testCases := []struct {
		input    string
		expected DurationList
		err      error
	}{
		{
			input:    "1s,2m,3h",
			expected: DurationList{time.Second, 2 * time.Minute, 3 * time.Hour},
			err:      nil,
		},
		{
			input:    "500ms,1s,2s",
			expected: DurationList{500 * time.Millisecond, time.Second, 2 * time.Second},
			err:      nil,
		},
		{
			input:    "100us,200us,300us",
			expected: DurationList{100 * time.Microsecond, 200 * time.Microsecond, 300 * time.Microsecond},
			err:      nil,
		},
		{
			input:    "1h,2h,3h",
			expected: DurationList{time.Hour, 2 * time.Hour, 3 * time.Hour},
			err:      nil,
		},
		{
			input:    "garbage",
			expected: nil,
			err:      fmt.Errorf("time: invalid duration \"garbage\""),
		},
	}

	for _, tc := range testCases {
		result, err := ParseDurationList(tc.input)
		if !reflect.DeepEqual(result, tc.expected) || !SameErrorMessages(err, tc.err) {
			t.Errorf("ParseList(%s) = %s, %v, expected %s, %v", tc.input, result, err, tc.expected, tc.err)
		}
	}
}
