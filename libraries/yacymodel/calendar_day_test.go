package yacymodel_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestCalendarDayOfDropsTimeOfDayAndZone(t *testing.T) {
	zone := time.FixedZone("east", 5*60*60)
	instant := time.Date(2025, time.February, 3, 2, 30, 0, 0, zone)

	got := yacymodel.CalendarDayOf(instant)
	want := yacymodel.NewCalendarDay(2025, time.February, 2)
	if got != want {
		t.Errorf("CalendarDayOf = %+v, want %+v", got, want)
	}
}

func TestNewCalendarDayNormalizesOutOfRangeParts(t *testing.T) {
	got := yacymodel.NewCalendarDay(2025, time.January, 32)
	want := yacymodel.NewCalendarDay(2025, time.February, 1)
	if got != want {
		t.Errorf("NewCalendarDay = %+v, want %+v", got, want)
	}
}

func TestCalendarDayTimeIsMidnightUTC(t *testing.T) {
	got := yacymodel.NewCalendarDay(2025, time.February, 3).Time()
	want := time.Date(2025, time.February, 3, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("Time = %v, want %v", got, want)
	}
}
