package vault

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var errTransactionNeverOpened = errors.New(
	"engine reported success without opening a transaction",
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
