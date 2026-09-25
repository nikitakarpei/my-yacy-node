// Package applog reports each URL metadata ask ceiling set under the most for a
// peer to the service log.
package applog

import (
	"context"
	"log/slog"
)

const msgAskCeilingSet = "url metadata ask ceiling of a peer set under the most"

type AskCeilingLog struct{}

func (AskCeilingLog) AskCeilingSet(
	ctx context.Context,
	address string,
	documentsPerSecond float64,
	ceiling int,
) {
	slog.DebugContext(ctx, msgAskCeilingSet,
		slog.String("address", address),
		slog.Float64("documentsPerSecond", documentsPerSecond),
		slog.Int("ceiling", ceiling),
	)
}

func (AskCeilingLog) AskCeilingHandedOut(context.Context, string, int) {}
