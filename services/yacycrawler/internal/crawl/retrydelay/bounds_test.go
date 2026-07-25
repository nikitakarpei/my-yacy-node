package retrydelay_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
)

func TestDelayGrowsFromFloorUpToCeiling(t *testing.T) {
	bounds := retrydelay.Bounds{Floor: time.Second, Ceiling: 4 * time.Second}

	for attempt, want := range map[int]time.Duration{
		1: time.Second,
		2: 1500 * time.Millisecond,
		3: 2250 * time.Millisecond,
		9: 4 * time.Second,
	} {
		if got := bounds.Delay(attempt); got != want {
			t.Fatalf("attempt %d: want %v, got %v", attempt, want, got)
		}
	}
}
