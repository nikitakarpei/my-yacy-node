package vault

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

type Txn struct {
	etx EngineTxn
}

func (v *Vault) Update(ctx context.Context, fn func(*Txn) error) error {
	if v == nil || v.engine == nil {
		return errVaultClosed
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}

	beganAt := time.Now()

	var timeline *writeTimeline

	var closureFailure error

	engineFailure := v.engine.Update(ctx, func(etx EngineTxn) error {
		timeline = newWriteTimeline()
		v.observer.ObserveWriteBegan(timeline.openedAt.Sub(beganAt))

		closureFailure = fn(&Txn{etx: etx})
		timeline.markExecuted()

		return closureFailure
	})
	if timeline != nil {
		timeline.markClosed()
	}

	if timeline == nil {
		if engineFailure == nil {
			engineFailure = errTransactionNeverOpened
		}

		reportWriteBeginRefused(ctx, v.observer, engineFailure)

		return wrapTxnError("begin write", engineFailure)
	}

	switch {
	case closureFailure != nil:
		reportWriteAborted(ctx, v.observer, timeline, closureFailure)

		return closureFailure
	case engineFailure != nil:
		reportWriteCommitRefused(ctx, v.observer, timeline, engineFailure)

		return wrapTxnError("write storage", engineFailure)
	default:
		reportWriteCommitted(ctx, v.observer, timeline)

		return nil
	}
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
) {
	observer.ObserveWriteCommitted(timeline.executeDuration(), timeline.closeDuration())
	slog.DebugContext(ctx, "write transaction committed")
}

func (v *Vault) View(ctx context.Context, fn func(*Txn) error) error {
	if v == nil || v.engine == nil {
		return errVaultClosed
	}

	v.observer.ObserveReadBegan()
	defer v.observer.ObserveReadEnded()

	return v.view(ctx, fn)
}

func (v *Vault) view(ctx context.Context, fn func(*Txn) error) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}

	if err := v.engine.View(ctx, func(etx EngineTxn) error {
		return fn(&Txn{etx: etx})
	}); err != nil {
		return wrapTxnError("read storage", err)
	}

	return nil
}

func wrapTxnError(operation string, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, errReadOnly) {
		return err
	}
	if errors.Is(err, ErrAtCapacity) {
		return err
	}

	return fmt.Errorf("%s: %w", operation, err)
}
