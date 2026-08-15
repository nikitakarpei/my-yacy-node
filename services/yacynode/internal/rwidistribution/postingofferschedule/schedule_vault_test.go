package postingofferschedule_test

import (
	"testing"
	"time"
)

func TestRescheduledOfferLeavesNoRowBehindAtAnyDueTime(t *testing.T) {
	for name, storedAt := range map[string]time.Time{
		"zero time":      {},
		"recent instant": time.Unix(1700000000, 123456789).UTC(),
		"far future":     time.Date(2500, time.January, 1, 0, 0, 0, 0, time.UTC),
	} {
		t.Run(name, func(t *testing.T) {
			offers := openOffers(t, storedAt)
			url := urlHash("u1")

			offers.store(t, testWord, url)
			offers.meetRedundancy(t, testWord, url)

			if due := offers.duePostings(t, 10); len(due) != 0 {
				t.Fatalf("due = %v, want none once the offer moved off %v", due, storedAt)
			}
		})
	}
}
