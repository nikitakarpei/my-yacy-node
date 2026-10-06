package pagereading

import (
	"math"
	"time"
)

type PageReadCutoff struct {
	PercentOfPages int
	Grace          time.Duration
}

type pageReadDeadlines struct {
	cutoff PageReadCutoff
	clock  Clock
}

func (deadlines pageReadDeadlines) deadlineFor(amountOfPagesWanted int) *pageReadDeadline {
	return &pageReadDeadline{
		clock: deadlines.clock,
		grace: deadlines.cutoff.Grace,
		amountOfPagesForTheGrace: int(math.Ceil(
			float64(deadlines.cutoff.PercentOfPages) / 100 * float64(amountOfPagesWanted),
		)),
		graceEndings: make(chan struct{}),
		stopGrace:    func() {},
	}
}
