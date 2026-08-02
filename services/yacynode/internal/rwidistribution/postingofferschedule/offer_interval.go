package postingofferschedule

import "time"

type OfferInterval struct {
	Shortest time.Duration
	Longest  time.Duration
}

func (i OfferInterval) widenedFrom(previousInterval time.Duration) time.Duration {
	return min(max(previousInterval*2, i.Shortest), i.Longest)
}
