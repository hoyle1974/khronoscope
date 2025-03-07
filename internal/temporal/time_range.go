package temporal

import (
	"time"

	"github.com/hoyle1974/khronoscope/internal/serializable"
)

type TimeRange struct {
	Min serializable.Time
	Max serializable.Time
}

func (t *TimeRange) Adjust(timestamp time.Time) {
	if t.Min.Time.IsZero() || timestamp.Before(t.Min.Time) {
		t.Min = serializable.NewTime(timestamp)
	}
	if t.Max.Time.IsZero() || timestamp.After(t.Max.Time) {
		t.Max = serializable.NewTime(timestamp)
	}
}

func (t *TimeRange) Contains(timestamp time.Time) bool {
	return (timestamp.Equal(t.Min.Time) || timestamp.After(t.Min.Time)) &&
		(timestamp.Equal(t.Max.Time) || timestamp.Before(t.Max.Time))
}
