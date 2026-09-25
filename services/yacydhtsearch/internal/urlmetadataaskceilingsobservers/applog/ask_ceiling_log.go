// Package applog reports each URL metadata ask ceiling set for a peer to the
// service log.
package applog

import (
	"context"
	"log/slog"
	"time"
)

const msgAskCeilingSet = "url metadata ask ceiling of a peer set"

type AskCeilingLog struct{}

func (AskCeilingLog) AskCeilingSet(
	ctx context.Context,
	address string,
	pace time.Duration,
	ceiling int,
) {
	slog.DebugContext(ctx, msgAskCeilingSet,
		slog.String("address", address),
		slog.Duration("pace", pace),
		slog.Int("ceiling", ceiling),
	)
}
