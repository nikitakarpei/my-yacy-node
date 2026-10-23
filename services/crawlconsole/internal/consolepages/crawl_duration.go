package consolepages

import (
	"time"
)

const hoursPerDay = 24

type crawlDuration struct {
	Days    int64
	Hours   int64
	Minutes int64
	Seconds int64
}

func crawlDurationOf(startedAt, finishedAt, now time.Time) crawlDuration {
	end := now
	if !finishedAt.IsZero() {
		end = finishedAt
	}
	elapsed := max(end.Sub(startedAt), 0)
	return crawlDuration{
		Days:    int64(elapsed / (hoursPerDay * time.Hour)),
		Hours:   int64(elapsed/time.Hour) % hoursPerDay,
		Minutes: int64(elapsed/time.Minute) % int64(time.Hour/time.Minute),
		Seconds: int64(elapsed/time.Second) % int64(time.Minute/time.Second),
	}
}
