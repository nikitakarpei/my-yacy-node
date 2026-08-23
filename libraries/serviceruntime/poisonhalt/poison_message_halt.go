// Package poisonhalt turns an undecodable broker message into an intake halt:
// it logs the message identity at ERROR and returns a sentinel error a consumer
// propagates to stop, leaving the message pending for an operator to resolve.
package poisonhalt

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

const msgIntakeHalted = "undecodable message halted intake"

var ErrPoisonMessage = errors.New("undecodable message halted intake")

func Halt(ctx context.Context, messageIdentity string, cause error) error {
	slog.ErrorContext(ctx, msgIntakeHalted,
		slog.String("message", messageIdentity),
		slog.Any("error", cause),
	)
	return fmt.Errorf("%w: %w", ErrPoisonMessage, cause)
}
