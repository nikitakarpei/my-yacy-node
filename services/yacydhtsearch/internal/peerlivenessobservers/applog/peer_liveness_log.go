// Package applog reports to the service log what one probe of one peer
// address found.
package applog

import (
	"context"
	"log/slog"
)

const (
	msgProbeCouldNotBeBuilt         = "a probe of a peer address could not be built"
	msgPeerDidNotAnswerTheProbe     = "a peer address did not answer the probe"
	msgPeerRefusedTheProbe          = "a peer address refused the probe"
	msgProbeAnswerCouldNotBeRead    = "the answer of a peer address could not be read"
	msgProbeAnswerCarriedNoRWICount = "the answer of a peer address carried no RWI count"
)

type PeerLivenessLog struct{}

func (PeerLivenessLog) ProbeCouldNotBeBuilt(ctx context.Context, address string, err error) {
	slog.DebugContext(ctx, msgProbeCouldNotBeBuilt,
		slog.String("address", address),
		slog.Any("error", err),
	)
}

func (PeerLivenessLog) PeerDidNotAnswerTheProbe(ctx context.Context, address string, err error) {
	slog.DebugContext(ctx, msgPeerDidNotAnswerTheProbe,
		slog.String("address", address),
		slog.Any("error", err),
	)
}

func (PeerLivenessLog) PeerRefusedTheProbe(ctx context.Context, address string, status int) {
	slog.DebugContext(ctx, msgPeerRefusedTheProbe,
		slog.String("address", address),
		slog.Int("status", status),
	)
}

func (PeerLivenessLog) ProbeAnswerCouldNotBeRead(ctx context.Context, address string, err error) {
	slog.DebugContext(ctx, msgProbeAnswerCouldNotBeRead,
		slog.String("address", address),
		slog.Any("error", err),
	)
}

func (PeerLivenessLog) ProbeAnswerCarriedNoRWICount(
	ctx context.Context,
	address string,
	err error,
) {
	slog.DebugContext(ctx, msgProbeAnswerCarriedNoRWICount,
		slog.String("address", address),
		slog.Any("error", err),
	)
}
