package vault

import (
	"context"
	"log/slog"
	"time"
)

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

func reportWriteBeginRefused(ctx context.Context, observer TransactionObserver, err error) {
	cause := refusalCauseOf(err)
	observer.ObserveWriteBeginRefused(cause)
	slog.ErrorContext(
		ctx,
		"write transaction refused before it began",
		slog.String("cause", string(cause)),
		slog.Any("error", err),
	)
}

func reportWriteAborted(
	ctx context.Context,
	observer TransactionObserver,
	timeline *writeTimeline,
	closureFailure error,
) {
	observer.ObserveWriteAborted(timeline.executeDuration(), timeline.closeDuration())
	slog.WarnContext(ctx, "write transaction aborted by caller", slog.Any("error", closureFailure))
}

func reportWriteCommitRefused(
	ctx context.Context,
	observer TransactionObserver,
	timeline *writeTimeline,
	engineFailure error,
) {
	cause := refusalCauseOf(engineFailure)
	observer.ObserveWriteCommitRefused(timeline.executeDuration(), timeline.closeDuration(), cause)
	slog.ErrorContext(
		ctx,
		"write transaction refused by storage engine",
		slog.String("cause", string(cause)),
		slog.Any("error", engineFailure),
	)
}

func reportWriteCommitted(
	ctx context.Context,
	observer TransactionObserver,
	timeline *writeTimeline,
	calledWriteOperation bool,
) {
	observer.ObserveWriteCommitted(
		timeline.executeDuration(),
		timeline.closeDuration(),
		calledWriteOperation,
	)
	slog.DebugContext(
		ctx,
		"write transaction committed",
		slog.Bool("calledWriteOperation", calledWriteOperation),
	)
}
