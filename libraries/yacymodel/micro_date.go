package yacymodel

import "time"

const microDateDay = 24 * time.Hour

// MicroDate is a day-resolution point in time, counted in whole days since
// the Unix epoch.
type MicroDate int

func MicroDateFromTime(t time.Time) MicroDate {
	return MicroDate(t.Unix() / int64(microDateDay/time.Second))
}

func (d MicroDate) Time() time.Time {
	return time.Unix(int64(d)*int64(microDateDay/time.Second), 0).UTC()
}
