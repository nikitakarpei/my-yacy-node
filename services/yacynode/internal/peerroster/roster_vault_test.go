package peerroster_test

import (
	"context"
	"testing"
	"time"
)

func TestRecentlyReachableOutsideTheNanosecondEpochRange(t *testing.T) {
	ctx := context.Background()

	for _, clockStart := range []time.Time{
		time.Date(1500, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, time.August, 9, 12, 0, 0, 1, time.UTC),
		time.Date(3000, time.January, 1, 0, 0, 0, 0, time.UTC),
	} {
		roster := openRoster(t, rosterFixture{
			reservoirCap:     8,
			reachableCap:     8,
			announceInterval: time.Minute,
			clockStart:       clockStart,
		})

		peer := seniorSeed(t, "peer", "203.0.113.1", 8090)
		roster.Discover(ctx, peer)
		roster.ConfirmReachable(ctx, peer)

		if !roster.IsRecentlyReachable(ctx, peer.Hash) {
			t.Errorf("peer confirmed reachable at %s is not recently reachable", clockStart)
		}
	}
}
