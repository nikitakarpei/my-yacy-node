package pullintake

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	msgAcknowledgeFailed = "message acknowledgement failed"
	msgReturnFailed      = "message not returned for redelivery"
)

type PendingMessage interface {
	Body() []byte
	Identity() string
	Acknowledge(ctx context.Context)
	Return(ctx context.Context)
	ReturnAfter(ctx context.Context, delay time.Duration)
}

type pendingMessage struct {
	message jetstream.Msg
}

func (m pendingMessage) Body() []byte {
	return m.message.Data()
}

func (m pendingMessage) Identity() string {
	metadata, err := m.message.Metadata()
	if err != nil {
		return m.message.Subject()
	}
	return fmt.Sprintf("%s %s/%d",
		m.message.Subject(), metadata.Stream, metadata.Sequence.Stream,
	)
}

func (m pendingMessage) Acknowledge(ctx context.Context) {
	if err := m.message.Ack(); err != nil {
		slog.WarnContext(ctx, msgAcknowledgeFailed,
			slog.String("message", m.Identity()),
			slog.Any("error", err),
		)
	}
}

func (m pendingMessage) Return(ctx context.Context) {
	if err := m.message.Nak(); err != nil {
		slog.WarnContext(ctx, msgReturnFailed,
			slog.String("message", m.Identity()),
			slog.Any("error", err),
		)
	}
}

func (m pendingMessage) ReturnAfter(ctx context.Context, delay time.Duration) {
	if err := m.message.NakWithDelay(delay); err != nil {
		slog.WarnContext(ctx, msgReturnFailed,
			slog.String("message", m.Identity()),
			slog.Duration("delay", delay),
			slog.Any("error", err),
		)
	}
}
