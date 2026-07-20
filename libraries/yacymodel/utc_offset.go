package yacymodel

import (
	"errors"
	"fmt"
	"time"
)

const (
	utcOffsetMinMinutes = -12 * 60
	utcOffsetMaxMinutes = 14 * 60
)

var ErrBadUTCOffset = errors.New("bad utc offset")

// UTCOffset is a peer's local timezone offset from UTC, measured in whole
// minutes east of UTC. It is a fixed position on the clock, not an elapsed span.
type UTCOffset struct {
	minutes int
}

func NewUTCOffset(minutesEast int) (UTCOffset, error) {
	if minutesEast < utcOffsetMinMinutes || minutesEast > utcOffsetMaxMinutes {
		return UTCOffset{}, fmt.Errorf("%w: %d minutes", ErrBadUTCOffset, minutesEast)
	}
	return UTCOffset{minutes: minutesEast}, nil
}

func UTCOffsetOf(t time.Time) UTCOffset {
	_, offsetSeconds := t.Zone()
	return UTCOffset{minutes: offsetSeconds / int(time.Minute/time.Second)}
}

func (o UTCOffset) MinutesEast() int { return o.minutes }
