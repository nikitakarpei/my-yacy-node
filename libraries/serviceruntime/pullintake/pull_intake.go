// Package pullintake runs bounded-concurrency intake from a JetStream pull
// consumer, halting on the first fatal message error and draining in-flight
// work before it returns. It pulls the next message only when a worker is free
// and its admission lets it, so it never holds a message it cannot work on.
package pullintake

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"golang.org/x/sync/errgroup"
)

var errAdmissionEnded = errors.New("admission ended")

type MessageSource interface {
	Next(...jetstream.FetchOpt) (jetstream.Msg, error)
}

type Process func(ctx context.Context, message PendingMessage) error

type Admission interface {
	AwaitAdmission(ctx context.Context) error
}

func Run(ctx context.Context, source MessageSource, messageConcurrency int, process Process) error {
	return RunAdmittedBy(ctx, source, messageConcurrency, everyMessageAdmitted{}, process)
}

func RunAdmittedBy(
	ctx context.Context,
	source MessageSource,
	messageConcurrency int,
	admission Admission,
	process Process,
) error {
	group, groupCtx := errgroup.WithContext(ctx)
	busyWorkers := make(chan struct{}, messageConcurrency)
	for {
		message, err := nextAdmittedMessage(groupCtx, source, busyWorkers, admission)
		if err != nil {
			return endOfIntake(ctx, group.Wait(), err)
		}
		group.Go(func() error {
			defer func() { <-busyWorkers }()
			return process(ctx, pendingMessage{message: message})
		})
	}
}

func nextAdmittedMessage(
	ctx context.Context,
	source MessageSource,
	busyWorkers chan<- struct{},
	admission Admission,
) (jetstream.Msg, error) {
	select {
	case busyWorkers <- struct{}{}:
	case <-ctx.Done():
		return nil, fmt.Errorf("await a free worker: %w", ctx.Err())
	}
	if err := admission.AwaitAdmission(ctx); err != nil {
		return nil, fmt.Errorf("%w: %w", errAdmissionEnded, err)
	}
	return nextMessage(ctx, source)
}

func nextMessage(ctx context.Context, source MessageSource) (jetstream.Msg, error) {
	for {
		message, err := source.Next(jetstream.FetchContext(contextWithoutDeadline{ctx}))
		if errors.Is(err, nats.ErrTimeout) || errors.Is(err, jetstream.ErrNoHeartbeat) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("next message: %w", err)
		}
		return message, nil
	}
}

type contextWithoutDeadline struct {
	context.Context
}

func (contextWithoutDeadline) Deadline() (time.Time, bool) {
	return time.Time{}, false
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
