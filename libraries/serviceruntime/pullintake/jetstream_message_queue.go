package pullintake

import (
	"context"
	"errors"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type MessageSource interface {
	Next(...jetstream.FetchOpt) (jetstream.Msg, error)
}

type jetStreamMessageQueue struct {
	source MessageSource
}

func MessageQueueOf(source MessageSource) MessageQueue {
	return jetStreamMessageQueue{source: source}
}

func (queue jetStreamMessageQueue) NextUpTo(ctx context.Context, _ int) ([]PendingMessage, error) {
	for {
		message, err := queue.source.Next(jetstream.FetchContext(contextWithoutDeadline{ctx}))
		if errors.Is(err, nats.ErrTimeout) || errors.Is(err, jetstream.ErrNoHeartbeat) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return []PendingMessage{pendingMessage{message: message}}, nil
	}
}

type contextWithoutDeadline struct {
	context.Context
}

func (contextWithoutDeadline) Deadline() (time.Time, bool) {
	return time.Time{}, false
}
