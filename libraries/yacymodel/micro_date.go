package yacymodel

import (
	"math"
	"time"
)

const microDateDay = 24 * time.Hour

type MicroDate uint16

func MicroDateOf(t time.Time) Optional[MicroDate] {
	return MicroDateFromDays(t.Unix() / int64(microDateDay/time.Second))
}

func MicroDateFromDays(days int64) Optional[MicroDate] {
	if days < 0 || days > math.MaxUint16 {
		return None[MicroDate]()
	}

	return Some(MicroDate(days))
}

func (d MicroDate) Time() time.Time {
	return time.Unix(int64(d)*int64(microDateDay/time.Second), 0).UTC()
}
