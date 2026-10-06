package pagereading

import (
	"sync"
	"time"
)

type pageReadDeadline struct {
	clock                    Clock
	grace                    time.Duration
	amountOfPagesForTheGrace int
	amountOfPagesSettled     int
	graceEndings             chan struct{}
	graceEnding              sync.Once
	stopGrace                func()
}

func (deadline *pageReadDeadline) pageSettled() {
	deadline.amountOfPagesSettled++
	if deadline.amountOfPagesSettled == deadline.amountOfPagesForTheGrace {
		deadline.stopGrace = deadline.clock.After(deadline.grace, deadline.endGrace)
	}
}

func (deadline *pageReadDeadline) endGrace() {
	deadline.graceEnding.Do(func() { close(deadline.graceEndings) })
}

func (deadline *pageReadDeadline) graceEnded() <-chan struct{} {
	return deadline.graceEndings
}

func (deadline *pageReadDeadline) stop() {
	deadline.stopGrace()
}
