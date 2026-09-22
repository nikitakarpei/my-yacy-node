// Package applog reports to the service log where the ledger of the peer
// judgements failed.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	msgJudgementDropped     = "the judgement of a peer was dropped because too many wait to be held"
	msgJudgementHoldFailed  = "the judgement of a peer could not be held"
	msgWatchFailed          = "the judgements of the peers could not be watched"
	msgJudgementUndecodable = "a judgement of a peer could not be decoded"
	msgWatchEnded           = "the watch of the judgements of the peers ended"
)

type JudgementLedgerLog struct{}

func (JudgementLedgerLog) JudgementDropped(
	ctx context.Context,
	peer yacymodel.Hash,
	question peerjudgements.Question,
) {
	slog.WarnContext(ctx, msgJudgementDropped,
		slog.String("peer", peer.String()),
		slog.String("question", string(question)),
	)
}

func (JudgementLedgerLog) JudgementHoldFailed(
	ctx context.Context,
	peer yacymodel.Hash,
	question peerjudgements.Question,
	err error,
) {
	slog.WarnContext(ctx, msgJudgementHoldFailed,
		slog.String("peer", peer.String()),
		slog.String("question", string(question)),
		slog.Any("error", err),
	)
}

func (JudgementLedgerLog) WatchFailed(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgWatchFailed, slog.Any("error", err))
}

func (JudgementLedgerLog) JudgementUndecodable(ctx context.Context, key string, err error) {
	slog.WarnContext(ctx, msgJudgementUndecodable,
		slog.String("key", key),
		slog.Any("error", err),
	)
}

func (JudgementLedgerLog) WatchEnded(ctx context.Context) {
	slog.WarnContext(ctx, msgWatchEnded)
}
