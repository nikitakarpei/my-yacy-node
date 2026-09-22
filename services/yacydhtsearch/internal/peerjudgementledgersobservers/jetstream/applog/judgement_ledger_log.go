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
	msgJudgementLookupFailed = "the judgement of a peer could not be read"
	msgJudgementHoldFailed   = "the judgement of a peer could not be held"
)

type JudgementLedgerLog struct{}

func (JudgementLedgerLog) JudgementLookupFailed(
	ctx context.Context,
	peer yacymodel.Hash,
	question peerjudgements.Question,
	err error,
) {
	slog.WarnContext(ctx, msgJudgementLookupFailed,
		slog.String("peer", peer.String()),
		slog.String("question", string(question)),
		slog.Any("error", err),
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
