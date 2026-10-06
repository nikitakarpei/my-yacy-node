package pagereading

import "time"

type pageReadBudget struct {
	duration time.Duration
	clock    Clock
}

func (budget pageReadBudget) startFor(page *startedPage) (stop func()) {
	return budget.clock.After(budget.duration, page.stopReading)
}
