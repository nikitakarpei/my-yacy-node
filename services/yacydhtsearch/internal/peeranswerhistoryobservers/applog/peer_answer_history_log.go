// Package applog reports the shared history of peer answers to the service log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswerhistory"
)

const (
	msgPeerAnswerAppendFailed = "the answer of a peer could not be appended to the history"
	msgPeerAnswerUnreadable   = "an answer in the history could not be read"
	msgPeerAnswerHistoryEnded = "the history of peer answers ended"
)

type PeerAnswerHistoryLog struct{}

func (PeerAnswerHistoryLog) PeerAnswerAppendFailed(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgPeerAnswerAppendFailed, slog.Any("error", err))
}

func (PeerAnswerHistoryLog) PeerAnswerUnreadable(
	ctx context.Context,
	position peeranswerhistory.AnswerPosition,
	err error,
) {
	slog.WarnContext(ctx, msgPeerAnswerUnreadable,
		slog.Uint64("position", uint64(position)),
		slog.Any("error", err),
	)
}

func (PeerAnswerHistoryLog) PeerAnswerHistoryEnded(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgPeerAnswerHistoryEnded, slog.Any("error", err))
}
