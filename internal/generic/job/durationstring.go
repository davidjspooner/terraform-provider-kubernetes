package job

import "time"

type DurationString struct {
	Duration time.Duration
}

func (v *DurationString) String() string {
	return v.Duration.String()
}

func (v *DurationString) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

func (v *DurationString) UnmarshalText(data []byte) error {
	duration, err := time.ParseDuration(string(data))
	if err != nil {
		return err
	}
	v.Duration = duration
	return nil
}
