// Package pullintake runs bounded-concurrency intake from a JetStream pull
// iterator, halting on the first fatal message error and draining in-flight
// work before it returns.
package pullintake

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"
	"golang.org/x/sync/errgroup"
)

type MessageSource interface {
	Messages(...jetstream.PullMessagesOpt) (jetstream.MessagesContext, error)
}

type Process func(ctx context.Context, message PendingMessage) error

func Run(ctx context.Context, source MessageSource, messageConcurrency int, process Process) error {
	iter, err := source.Messages()
	if err != nil {
		return fmt.Errorf("open message iterator: %w", err)
	}
	defer iter.Stop()

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(messageConcurrency)
	go stopIteratorOnCancel(groupCtx, iter)

	for {
		msg, err := iter.Next()
		if err != nil {
			if waitErr := group.Wait(); waitErr != nil {
				return waitErr
			}
			if errors.Is(err, jetstream.ErrMsgIteratorClosed) || ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("next message: %w", err)
		}
		group.Go(func() error { return process(ctx, pendingMessage{message: msg}) })
	}
}

func stopIteratorOnCancel(ctx context.Context, iter jetstream.MessagesContext) {
	<-ctx.Done()
	iter.Stop()
}
