// Package pullintake runs bounded-concurrency intake from a source of pending
// messages, such as a JetStream pull consumer, halting on the first fatal message
// error and draining in-flight work before it returns. It takes the next message
// only when a worker is free and its admission lets it, so it never holds a
// message it cannot work on.
package pullintake

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
	"golang.org/x/sync/errgroup"
)

var errAdmissionEnded = errors.New("admission ended")

type PendingMessages interface {
	Next(ctx context.Context) (PendingMessage, error)
}

type Process func(ctx context.Context, message PendingMessage) error

type Admission interface {
	AwaitAdmission(ctx context.Context) error
}

func Run(
	ctx context.Context,
	messages PendingMessages,
	messageConcurrency int,
	process Process,
) error {
	return RunAdmittedBy(ctx, messages, messageConcurrency, everyMessageAdmitted{}, process)
}

func RunAdmittedBy(
	ctx context.Context,
	messages PendingMessages,
	messageConcurrency int,
	admission Admission,
	process Process,
) error {
	group, groupCtx := errgroup.WithContext(ctx)
	busyWorkers := make(chan struct{}, messageConcurrency)
	for {
		message, err := nextAdmittedMessage(groupCtx, messages, busyWorkers, admission)
		if err != nil {
			return endOfIntake(ctx, group.Wait(), err)
		}
		group.Go(func() error {
			defer func() { <-busyWorkers }()
			return process(ctx, message)
		})
	}
}

func nextAdmittedMessage(
	ctx context.Context,
	messages PendingMessages,
	busyWorkers chan<- struct{},
	admission Admission,
) (PendingMessage, error) {
	select {
	case busyWorkers <- struct{}{}:
	case <-ctx.Done():
		return nil, fmt.Errorf("await a free worker: %w", ctx.Err())
	}
	if err := admission.AwaitAdmission(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", errAdmissionEnded, err)
	}
	message, err := messages.Next(ctx)
	if err != nil {
		return nil, fmt.Errorf("next message: %w", err)
	}
	return message, nil
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

type everyMessageAdmitted struct{}

func (everyMessageAdmitted) AwaitAdmission(context.Context) error {
	return nil
}
