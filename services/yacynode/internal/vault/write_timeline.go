package vault

import "time"

type writeTimeline struct {
	openedAt   time.Time
	executedAt time.Time
	closedAt   time.Time
}

func newWriteTimeline() *writeTimeline {
	return &writeTimeline{openedAt: time.Now()}
}

func (t *writeTimeline) markExecuted() {
	t.executedAt = time.Now()
}

func (t *writeTimeline) markClosed() {
	t.closedAt = time.Now()
}

func (t *writeTimeline) executeDuration() time.Duration {
	return t.executedAt.Sub(t.openedAt)
}

func (t *writeTimeline) closeDuration() time.Duration {
	return t.closedAt.Sub(t.executedAt)
}
