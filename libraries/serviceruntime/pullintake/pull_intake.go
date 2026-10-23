// Package pullintake runs bounded-concurrency intake from a source of pending
// items, such as the messages of a JetStream pull consumer, halting on the first
// fatal error and draining in-flight work before it returns. It takes the next item
// only when a worker is free and its admission lets it, so it never holds an
// item it cannot work on.
package pullintake

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
	"golang.org/x/sync/errgroup"
)

var errAdmissionEnded = errors.New("admission ended")

type Pending[Item any] interface {
	Next(ctx context.Context) (Item, error)
}

type PendingMessages = Pending[PendingMessage]

type Admission interface {
	AwaitAdmission(ctx context.Context) error
}

func Run[Item any](
	ctx context.Context,
	items Pending[Item],
	concurrency int,
	process func(ctx context.Context, item Item) error,
) error {
	return RunAdmittedBy(ctx, items, concurrency, everyItemAdmitted{}, process)
}

func RunAdmittedBy[Item any](
	ctx context.Context,
	items Pending[Item],
	concurrency int,
	admission Admission,
	process func(ctx context.Context, item Item) error,
) error {
	group, groupCtx := errgroup.WithContext(ctx)
	busyWorkers := make(chan struct{}, concurrency)
	for {
		item, err := nextAdmittedItem(groupCtx, items, busyWorkers, admission)
		if err != nil {
			return endOfIntake(ctx, group.Wait(), err)
		}
		group.Go(func() error {
			defer func() { <-busyWorkers }()
			return process(ctx, item)
		})
	}
}

func nextAdmittedItem[Item any](
	ctx context.Context,
	items Pending[Item],
	busyWorkers chan<- struct{},
	admission Admission,
) (Item, error) {
	var none Item
	select {
	case busyWorkers <- struct{}{}:
	case <-ctx.Done():
		return none, fmt.Errorf("await a free worker: %w", ctx.Err())
	}
	if err := admission.AwaitAdmission(ctx); err != nil {
		return none, fmt.Errorf("%w: %w", errAdmissionEnded, err)
	}
	item, err := items.Next(ctx)
	if err != nil {
		return none, fmt.Errorf("next item: %w", err)
	}
	return item, nil
}

func endOfIntake(ctx context.Context, processErr error, intakeErr error) error {
	if processErr != nil {
		return processErr
	}
	if ctx.Err() != nil || errors.Is(intakeErr, errAdmissionEnded) ||
		errors.Is(intakeErr, nats.ErrConnectionClosed) {
		return nil //nolint:nilerr // an ended context, admission or connection ends intake cleanly
	}
	return intakeErr
}

type everyItemAdmitted struct{}

func (everyItemAdmitted) AwaitAdmission(context.Context) error {
	return nil
}
