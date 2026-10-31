package yacymodel_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestMicroDateTimeRoundTrip(t *testing.T) {
	day := time.Date(2026, time.July, 18, 0, 0, 0, 0, time.UTC)
	date, ok := yacymodel.MicroDateOf(day).Get()
	if !ok {
		t.Fatalf("MicroDateOf(%v) is absent", day)
	}
	if got := date.Time(); !got.Equal(day) {
		t.Fatalf("Time() = %v, want %v", got, day)
	}
}

func TestMicroDateHoldsTheDaysTwoBytesCount(t *testing.T) {
	t.Parallel()

	for days, want := range map[int64]yacymodel.Optional[yacymodel.MicroDate]{
		0:     yacymodel.Some[yacymodel.MicroDate](0),
		65535: yacymodel.Some[yacymodel.MicroDate](65535),
		65536: yacymodel.None[yacymodel.MicroDate](),
		-1:    yacymodel.None[yacymodel.MicroDate](),
	} {
		if got := yacymodel.MicroDateFromDays(days); got != want {
			t.Errorf("micro date from %d days = %+v, want %+v", days, got, want)
		}
	}
}
