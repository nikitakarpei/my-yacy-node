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
	// TECHDEBT: collaborators rule — other units call tx.etx.X(bucket).Y() and set its fields
	etx                  EngineTxn
	at                   time.Time
	calledWriteOperation bool
	afterCommit          []func()
	pendingWork          []pendingWork
	pendingWorkByOwner   map[any]pendingWork
}

type pendingWork interface {
	finishWithin(etx EngineTxn) error
}

func (t *Txn) RunAfterCommit(run func()) {
	t.afterCommit = append(t.afterCommit, run)
}

func (t *Txn) pendingWorkFor[W pendingWork](owner any, newWork func() W) W {
	if work, found := t.pendingWorkByOwner[owner]; found {
		return work.(W)
	}
	work := newWork()
	if t.pendingWorkByOwner == nil {
		t.pendingWorkByOwner = map[any]pendingWork{}
	}
	t.pendingWorkByOwner[owner] = work
	t.pendingWork = append(t.pendingWork, work)

	return work
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

	var beginReported bool

	var closureFailure error

	var calledWriteOperation bool

	var afterCommit []func()

	engineFailure := v.engine.Update(ctx, func(etx EngineTxn) error {
		timeline = newWriteTimeline()
		if !beginReported {
			v.transactionObserver.ObserveWriteBegan(timeline.openedAt.Sub(beganAt))
			beginReported = true
		}

		tx := &Txn{etx: etx, at: v.clock.Now()}
		closureFailure = tx.run(fn)
		calledWriteOperation = tx.calledWriteOperation
		afterCommit = tx.afterCommit
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

		reportWriteBeginRefused(ctx, v.transactionObserver, engineFailure)

		return wrapTxnError("begin write", engineFailure)
	}

	switch {
	case closureFailure != nil:
		reportWriteAborted(ctx, v.transactionObserver, timeline, closureFailure)

		return closureFailure
	case engineFailure != nil:
		reportWriteCommitRefused(ctx, v.transactionObserver, timeline, engineFailure)

		return wrapTxnError("write storage", engineFailure)
	default:
		reportWriteCommitted(ctx, v.transactionObserver, timeline, calledWriteOperation)
		runEach(afterCommit)

		return nil
	}
}

func (t *Txn) run(fn func(*Txn) error) error {
	if err := fn(t); err != nil {
		return err
	}

	return t.finishPendingWork()
}

func (t *Txn) finishPendingWork() error {
	for _, work := range t.pendingWork {
		if err := work.finishWithin(t.etx); err != nil {
			return err
		}
	}

	return nil
}

func (v *Vault) View(ctx context.Context, fn func(*Txn) error) error {
	if v == nil || v.engine == nil {
		return errVaultClosed
	}

	v.transactionObserver.ObserveReadBegan()
	defer v.transactionObserver.ObserveReadEnded()

	return v.view(ctx, fn)
}

func (v *Vault) view(ctx context.Context, fn func(*Txn) error) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context: %w", err)
	}

	if err := v.engine.View(ctx, func(etx EngineTxn) error {
		return fn(&Txn{etx: etx, at: v.clock.Now()})
	}); err != nil {
		return wrapTxnError("read storage", err)
	}

	return nil
}

func runEach(callbacks []func()) {
	for _, run := range callbacks {
		run()
	}
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
