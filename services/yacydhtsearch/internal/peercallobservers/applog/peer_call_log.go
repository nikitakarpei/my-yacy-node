// Package applog reports the outcome of each peer call, and what the peer call
// asked the peer for, to the log.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

const (
	msgPeerCallWaitsForASlot       = "peer call waits for an in-flight slot"
	msgPeerCallTookASlot           = "peer call took an in-flight slot"
	msgPeerCallCancelled           = "peer call was cancelled before the peer answered"
	msgPeerAnsweredURLMetadata     = "peer answered the metadata it holds for the documents"
	msgPeerAnsweredSearchDocuments = "peer answered the documents it lists and matches for a word"
	msgPeerRefused                 = "peer refused a search"
	msgPeerUnreachable             = "peer could not be reached for a search"
	msgPeerAnswerUnreadable        = "peer answered a search unreadably"
)

type PeerCallLog struct{}

func (PeerCallLog) PeerCallWaitsForASlot(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
) {
	slog.DebugContext(ctx, msgPeerCallWaitsForASlot,
		slog.String("address", address),
		slog.String("askedFor", string(askedFor)),
	)
}

func (PeerCallLog) PeerCallTookASlot(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	waited time.Duration,
) {
	slog.DebugContext(ctx, msgPeerCallTookASlot,
		slog.String("address", address),
		slog.String("askedFor", string(askedFor)),
		slog.Duration("waited", waited),
	)
}

func (PeerCallLog) PeerAnsweredURLMetadata(
	ctx context.Context,
	address string,
	amountOfDescribedDocuments int,
	spent time.Duration,
) {
	slog.DebugContext(ctx, msgPeerAnsweredURLMetadata,
		slog.String("address", address),
		slog.Int("amountOfDescribedDocuments", amountOfDescribedDocuments),
		slog.Duration("spent", spent),
	)
}

func (PeerCallLog) PeerAnsweredSearchDocuments(
	ctx context.Context,
	address string,
	amountOfListedDocuments int,
	amountOfMatchedDocuments int,
	spent time.Duration,
) {
	slog.DebugContext(ctx, msgPeerAnsweredSearchDocuments,
		slog.String("address", address),
		slog.Int("amountOfListedDocuments", amountOfListedDocuments),
		slog.Int("amountOfMatchedDocuments", amountOfMatchedDocuments),
		slog.Duration("spent", spent),
	)
}

func (PeerCallLog) PeerRefused(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	status int,
	spent time.Duration,
) {
	slog.WarnContext(ctx, msgPeerRefused,
		slog.String("address", address),
		slog.String("askedFor", string(askedFor)),
		slog.Int("status", status),
		slog.Duration("spent", spent),
	)
}

func (PeerCallLog) PeerUnreachable(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	cause error,
	spent time.Duration,
) {
	slog.WarnContext(ctx, msgPeerUnreachable,
		slog.String("address", address),
		slog.String("askedFor", string(askedFor)),
		slog.Any("error", cause),
		slog.Duration("spent", spent),
	)
}

func (PeerCallLog) PeerAnswerUnreadable(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	cause error,
	spent time.Duration,
) {
	slog.WarnContext(ctx, msgPeerAnswerUnreadable,
		slog.String("address", address),
		slog.String("askedFor", string(askedFor)),
		slog.Any("error", cause),
		slog.Duration("spent", spent),
	)
}

func (PeerCallLog) PeerCallCancelled(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	spent time.Duration,
) {
	slog.DebugContext(ctx, msgPeerCallCancelled,
		slog.String("address", address),
		slog.String("askedFor", string(askedFor)),
		slog.Duration("spent", spent),
	)
}
