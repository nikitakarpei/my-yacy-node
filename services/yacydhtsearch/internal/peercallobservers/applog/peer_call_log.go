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
	msgPeerAnsweredMatchedItems  = "peer answered the items it matched"
	msgPeerAnsweredURLMetadata   = "peer answered the metadata it holds for the documents"
	msgPeerAnsweredHeldDocuments = "peer answered the documents it holds for a word"
	msgPeerRefused               = "peer refused a search"
	msgPeerUnreachable           = "peer could not be reached for a search"
	msgPeerAnswerUnreadable      = "peer answered a search unreadably"
)

type PeerCallLog struct{}

func (PeerCallLog) PeerAnsweredMatchedItems(
	ctx context.Context,
	address string,
	amountOfMatchedItems int,
	spent time.Duration,
) {
	slog.DebugContext(ctx, msgPeerAnsweredMatchedItems,
		slog.String("address", address),
		slog.Int("amountOfMatchedItems", amountOfMatchedItems),
		slog.Duration("spent", spent),
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

func (PeerCallLog) PeerAnsweredHeldDocuments(
	ctx context.Context,
	address string,
	amountOfDocuments int,
	spent time.Duration,
) {
	slog.DebugContext(ctx, msgPeerAnsweredHeldDocuments,
		slog.String("address", address),
		slog.Int("amountOfDocuments", amountOfDocuments),
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
