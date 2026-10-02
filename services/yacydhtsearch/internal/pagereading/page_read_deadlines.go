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
	budget time.Duration
	cutoff PageReadCutoff
	clock  Clock
}

func (deadlines pageReadDeadlines) deadlineFor(amountOfPagesWanted int) *pageReadDeadline {
	deadline := &pageReadDeadline{
		clock: deadlines.clock,
		grace: deadlines.cutoff.Grace,
		amountOfPagesForTheGrace: int(math.Ceil(
			float64(deadlines.cutoff.PercentOfPages) / 100 * float64(amountOfPagesWanted),
		)),
		graceEndings:  make(chan struct{}),
		budgetEndings: make(chan struct{}),
		stopGrace:     func() {},
	}
	deadline.startBudget(deadlines.budget)

	return deadline
}
