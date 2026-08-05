package postingofferschedule

import "time"

type OfferInterval struct {
	Shortest time.Duration
	Longest  time.Duration
}

func (i OfferInterval) widenedFrom(previousInterval time.Duration) time.Duration {
	return min(max(previousInterval*2, i.Shortest), i.Longest)
}

func (i OfferInterval) nextDueAtFrom(previousDueAt time.Time, now time.Time) time.Time {
	missedIntervals := max(now.Sub(previousDueAt)/i.Longest, 0)

	return previousDueAt.Add((missedIntervals + 1) * i.Longest)
}
