package rwidistribution

import "time"

type postingOfferCadence struct {
	refresh time.Duration
	retry   time.Duration
}

func (c postingOfferCadence) NextDue(
	now time.Time,
	replicated bool,
	retryAfter time.Duration,
) time.Time {
	if replicated {
		return now.Add(c.refresh)
	}
	if retryAfter <= 0 {
		retryAfter = c.retry
	}

	return now.Add(retryAfter)
}
