package yacymodel

import "time"

// CalendarDay is a date in UTC, without a time of day.
type CalendarDay struct {
	Year  int
	Month time.Month
	Day   int
}

func NewCalendarDay(year int, month time.Month, day int) CalendarDay {
	return CalendarDayOf(time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
}

func CalendarDayOf(instant time.Time) CalendarDay {
	year, month, day := instant.UTC().Date()

	return CalendarDay{Year: year, Month: month, Day: day}
}

func (d CalendarDay) IsZero() bool {
	return d == CalendarDay{}
}

func (d CalendarDay) Time() time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)
}
