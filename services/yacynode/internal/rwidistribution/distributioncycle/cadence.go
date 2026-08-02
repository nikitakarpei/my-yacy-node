package distributioncycle

import "time"

type Cadence struct {
	Refresh time.Duration
	Backoff time.Duration
}

func (c Cadence) NextDue(
	now time.Time,
	redundancyMet bool,
	peerBackoff time.Duration,
) time.Time {
	if redundancyMet {
		return now.Add(c.Refresh)
	}
	if peerBackoff <= 0 {
		return now.Add(c.Backoff)
	}

	return now.Add(peerBackoff)
}
