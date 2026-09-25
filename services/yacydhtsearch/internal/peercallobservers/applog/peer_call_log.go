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
	msgPeerCallWaitsForASlot   = "peer call waits for an in-flight slot"
	msgPeerCallTookASlot       = "peer call took an in-flight slot"
	msgPeerCallCancelled       = "peer call was cancelled before the peer answered"
	msgPeerAnsweredURLMetadata = "peer answered the metadata it holds for the documents"
	msgPeerSearchedDocuments   = "peer searched its documents for the word"
	msgPeerRefused             = "peer refused a search"
	msgPeerUnreachable         = "peer could not be reached for a search"
	msgPeerAnswerUnreadable    = "peer answered a search unreadably"
	msgPeerHeadersLate         = "peer sent no headers within the headers timeout"
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
	amountOfDocumentsAsked int,
	amountOfDescribedDocuments int,
	spent time.Duration,
) {
	slog.DebugContext(ctx, msgPeerAnsweredURLMetadata,
		slog.String("address", address),
		slog.Int("amountOfDocumentsAsked", amountOfDocumentsAsked),
		slog.Int("amountOfDescribedDocuments", amountOfDescribedDocuments),
		slog.Duration("spent", spent),
	)
}

func (PeerCallLog) PeerSearchedDocuments(
	ctx context.Context,
	address string,
	amountOfDocumentsInTheAbstract int,
	amountOfMatchedDocuments int,
	spent time.Duration,
) {
	slog.DebugContext(ctx, msgPeerSearchedDocuments,
		slog.String("address", address),
		slog.Int("amountOfDocumentsInTheAbstract", amountOfDocumentsInTheAbstract),
		slog.Int("amountOfMatchedDocuments", amountOfMatchedDocuments),
		slog.Duration("spent", spent),
	)
}

//nolint:revive // argument-limit: an outcome names its peer, ask, documents asked, failure and time
func (PeerCallLog) PeerRefused(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	amountOfDocumentsAsked int,
	status int,
	spent time.Duration,
) {
	slog.WarnContext(ctx, msgPeerRefused,
		slog.String("address", address),
		slog.String("askedFor", string(askedFor)),
		slog.Int("amountOfDocumentsAsked", amountOfDocumentsAsked),
		slog.Int("status", status),
		slog.Duration("spent", spent),
	)
}

//nolint:revive // argument-limit: an outcome names its peer, ask, documents asked, failure and time
func (PeerCallLog) PeerUnreachable(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	amountOfDocumentsAsked int,
	cause error,
	spent time.Duration,
) {
	slog.WarnContext(ctx, msgPeerUnreachable,
		slog.String("address", address),
		slog.String("askedFor", string(askedFor)),
		slog.Int("amountOfDocumentsAsked", amountOfDocumentsAsked),
		slog.Any("error", cause),
		slog.Duration("spent", spent),
	)
}

//nolint:revive // argument-limit: an outcome names its peer, ask, documents asked, failure and time
func (PeerCallLog) PeerAnswerUnreadable(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	amountOfDocumentsAsked int,
	cause error,
	spent time.Duration,
) {
	slog.WarnContext(ctx, msgPeerAnswerUnreadable,
		slog.String("address", address),
		slog.String("askedFor", string(askedFor)),
		slog.Int("amountOfDocumentsAsked", amountOfDocumentsAsked),
		slog.Any("error", cause),
		slog.Duration("spent", spent),
	)
}

func (PeerCallLog) PeerHeadersLate(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	amountOfDocumentsAsked int,
	spent time.Duration,
) {
	slog.WarnContext(ctx, msgPeerHeadersLate,
		slog.String("address", address),
		slog.String("askedFor", string(askedFor)),
		slog.Int("amountOfDocumentsAsked", amountOfDocumentsAsked),
		slog.Duration("spent", spent),
	)
}

func (PeerCallLog) PeerCallCancelled(
	ctx context.Context,
	address string,
	askedFor peerasks.AskedFor,
	amountOfDocumentsAsked int,
	spent time.Duration,
) {
	slog.DebugContext(ctx, msgPeerCallCancelled,
		slog.String("address", address),
		slog.String("askedFor", string(askedFor)),
		slog.Int("amountOfDocumentsAsked", amountOfDocumentsAsked),
		slog.Duration("spent", spent),
	)
}
