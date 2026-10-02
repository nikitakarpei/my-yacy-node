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
	budgetEndings            chan struct{}
	graceEnding              sync.Once
	budgetEnding             sync.Once
	stopGrace                func()
	stopBudget               func()
}

func (deadline *pageReadDeadline) startBudget(budget time.Duration) {
	deadline.stopBudget = deadline.clock.After(budget, deadline.endBudget)
}

func (deadline *pageReadDeadline) endBudget() {
	deadline.budgetEnding.Do(func() { close(deadline.budgetEndings) })
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

func (deadline *pageReadDeadline) budgetEnded() <-chan struct{} {
	return deadline.budgetEndings
}

func (deadline *pageReadDeadline) stop() {
	deadline.stopGrace()
	deadline.stopBudget()
}
