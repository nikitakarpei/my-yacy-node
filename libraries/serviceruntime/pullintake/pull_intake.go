// Package pullintake runs bounded-concurrency intake from a queue of items, such as
// the messages of a JetStream pull consumer, halting on the first fatal error and
// draining in-flight work before it returns. It takes the next items only when a
// worker is free and its admission lets it, so it never holds an item it cannot work
// on. The queue is asked for as many items as there are free workers; it may give fewer.
package pullintake

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
	"golang.org/x/sync/errgroup"
)

var errAdmissionEnded = errors.New("admission ended")

type Queue[Item any] interface {
	NextUpTo(ctx context.Context, amount int) ([]Item, error)
}

type MessageQueue = Queue[PendingMessage]

type Admission interface {
	AwaitAdmission(ctx context.Context) error
}

func Run[Item any](
	ctx context.Context,
	queue Queue[Item],
	concurrency int,
	process func(ctx context.Context, item Item) error,
) error {
	return RunAdmittedBy(ctx, queue, concurrency, everyItemAdmitted{}, process)
}

func RunAdmittedBy[Item any](
	ctx context.Context,
	queue Queue[Item],
	concurrency int,
	admission Admission,
	process func(ctx context.Context, item Item) error,
) error {
	group, groupCtx := errgroup.WithContext(ctx)
	busyWorkers := make(chan struct{}, concurrency)
	for {
		admittedItems, err := nextAdmittedItems(groupCtx, queue, busyWorkers, admission)
		if err != nil {
			return endOfIntake(ctx, group.Wait(), err)
		}
		for _, item := range admittedItems {
			group.Go(func() error {
				defer func() { <-busyWorkers }()
				return process(ctx, item)
			})
		}
	}
}

func nextAdmittedItems[Item any](
	ctx context.Context,
	queue Queue[Item],
	busyWorkers chan struct{},
	admission Admission,
) ([]Item, error) {
	select {
	case busyWorkers <- struct{}{}:
	case <-ctx.Done():
		return nil, fmt.Errorf("await a free worker: %w", ctx.Err())
	}
	if err := admission.AwaitAdmission(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", errAdmissionEnded, err)
	}
	takenWorkers := 1 + takeFreeWorkers(busyWorkers)
	nextItems, err := queue.NextUpTo(ctx, takenWorkers)
	if err != nil {
		return nil, fmt.Errorf("next items: %w", err)
	}
	for range takenWorkers - len(nextItems) {
		<-busyWorkers
	}
	return nextItems, nil
}

func takeFreeWorkers(busyWorkers chan<- struct{}) int {
	takenWorkers := 0
	for {
		select {
		case busyWorkers <- struct{}{}:
			takenWorkers++
		default:
			return takenWorkers
		}
	}
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
