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

type jetStreamPendingMessages struct {
	source MessageSource
}

func PendingMessagesOf(source MessageSource) PendingMessages {
	return jetStreamPendingMessages{source: source}
}

func (m jetStreamPendingMessages) Next(ctx context.Context) (PendingMessage, error) {
	for {
		message, err := m.source.Next(jetstream.FetchContext(contextWithoutDeadline{ctx}))
		if errors.Is(err, nats.ErrTimeout) || errors.Is(err, jetstream.ErrNoHeartbeat) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return pendingMessage{message: message}, nil
	}
}

type contextWithoutDeadline struct {
	context.Context
}

func (contextWithoutDeadline) Deadline() (time.Time, bool) {
	return time.Time{}, false
}
