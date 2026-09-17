// Package applog reports to the service log how near the peers a query asks
// sit to the places on the DHT ring where the postings of its words belong.
package applog

import (
	"context"
	"log/slog"
)

const msgPeersTakenFromTheRing = "peers were taken from the ring for a query word"

type PeerChoiceLog struct{}

func (PeerChoiceLog) PeersTakenFromTheRing(ctx context.Context, ringFractions []float64) {
	slog.DebugContext(ctx, msgPeersTakenFromTheRing,
		slog.Int("peers", len(ringFractions)),
		slog.Any("ringFractions", ringFractions),
	)
}
