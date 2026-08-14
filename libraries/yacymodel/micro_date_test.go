package yacymodel_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestMicroDateTimeRoundTrip(t *testing.T) {
	day := time.Date(2026, time.July, 18, 0, 0, 0, 0, time.UTC)
	got := yacymodel.MicroDateFromTime(day).Time()
	if !got.Equal(day) {
		t.Fatalf("Time() = %v, want %v", got, day)
	}
}
