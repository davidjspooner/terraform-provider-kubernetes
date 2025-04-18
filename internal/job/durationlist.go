package job

import (
	"fmt"
	"strings"
	"time"
)

type DurationList []time.Duration

func ParseDurationList(s string) (DurationList, error) {
	if strings.TrimSpace(s) == "" {
		return nil, fmt.Errorf("duration list cannot be empty or whitespace")
	}

	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' '
	})
	list := make(DurationList, len(parts))
	for i, part := range parts {
		duration, err := time.ParseDuration(part)
		if err != nil {
			return nil, fmt.Errorf("invalid duration %q: %v", part, err)
		}
		list[i] = duration
	}
	return list, nil
}

func (list DurationList) String() string {
	if len(list) == 0 {
		return "<undefined>"
	}
	sb := strings.Builder{}
	for i, d := range list {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(d.String())
	}
	return sb.String()
}

func (list DurationList) MarshalText() ([]byte, error) {
	return []byte(list.String()), nil
}

func (list *DurationList) UnmarshalText(data []byte) error {
	l, err := ParseDurationList(string(data))
	if err != nil {
		return err
	}
	*list = l
	return nil
}
