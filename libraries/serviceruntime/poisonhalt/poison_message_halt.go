// Package poisonhalt turns an undecodable broker message into an intake halt:
// it logs the message identity at ERROR and returns a sentinel error a consumer
// propagates to stop, leaving the message pending for an operator to resolve.
package poisonhalt

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"
)

const msgIntakeHalted = "undecodable message halted intake"

var ErrPoisonMessage = errors.New("undecodable message halted intake")

func Halt(ctx context.Context, msg jetstream.Msg, cause error) error {
	attrs := []any{
		slog.String("subject", msg.Subject()),
		slog.Any("error", cause),
	}
	if metadata, err := msg.Metadata(); err == nil {
		attrs = append(attrs,
			slog.String("stream", metadata.Stream),
			slog.Uint64("streamSequence", metadata.Sequence.Stream),
		)
	}
	slog.ErrorContext(ctx, msgIntakeHalted, attrs...)
	return fmt.Errorf("%w: %w", ErrPoisonMessage, cause)
}
