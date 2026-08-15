// Package postingofferinterval holds how long a stored posting waits between
// distribution offers. The wait starts at the shortest interval, doubles on
// every missed redundancy up to the longest, and returns to the shortest once
// redundancy is met. A posting whose redundancy is met keeps offering at the
// longest interval, anchored at the time its previous offer fell due.
package postingofferinterval

import "time"

type Bounds struct {
	Shortest time.Duration
	Longest  time.Duration
}

func (b Bounds) WidenedFrom(previousInterval time.Duration) time.Duration {
	return min(max(previousInterval*2, b.Shortest), b.Longest)
}

func (b Bounds) PauseFrom(
	previousInterval time.Duration,
	requestedPause time.Duration,
) time.Duration {
	return max(b.WidenedFrom(previousInterval), requestedPause)
}

func (b Bounds) NextOfferDueFrom(previousDueAt time.Time, now time.Time) time.Time {
	missedOffers := max(now.Sub(previousDueAt)/b.Longest, 0)

	return previousDueAt.Add((missedOffers + 1) * b.Longest)
}
