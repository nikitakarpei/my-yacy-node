package jetstream

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type JudgementLedgerObserver interface {
	JudgementLookupFailed(
		ctx context.Context,
		peer yacymodel.Hash,
		question peerjudgements.Question,
		err error,
	)
	JudgementHoldFailed(
		ctx context.Context,
		peer yacymodel.Hash,
		question peerjudgements.Question,
		err error,
	)
}

type JudgementLedgerObservers []JudgementLedgerObserver

func (observers JudgementLedgerObservers) JudgementLookupFailed(
	ctx context.Context,
	peer yacymodel.Hash,
	question peerjudgements.Question,
	err error,
) {
	for _, observer := range observers {
		observer.JudgementLookupFailed(ctx, peer, question, err)
	}
}

func (observers JudgementLedgerObservers) JudgementHoldFailed(
	ctx context.Context,
	peer yacymodel.Hash,
	question peerjudgements.Question,
	err error,
) {
	for _, observer := range observers {
		observer.JudgementHoldFailed(ctx, peer, question, err)
	}
}
