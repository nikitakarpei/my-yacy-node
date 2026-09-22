package jetstream

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type JudgementLedgerObserver interface {
	JudgementDropped(ctx context.Context, peer yacymodel.Hash, question peerjudgements.Question)
	JudgementHoldFailed(
		ctx context.Context,
		peer yacymodel.Hash,
		question peerjudgements.Question,
		err error,
	)
	WatchFailed(ctx context.Context, err error)
	JudgementUndecodable(ctx context.Context, key string, err error)
	WatchEnded(ctx context.Context)
}

type JudgementLedgerObservers []JudgementLedgerObserver

func (observers JudgementLedgerObservers) JudgementDropped(
	ctx context.Context,
	peer yacymodel.Hash,
	question peerjudgements.Question,
) {
	for _, observer := range observers {
		observer.JudgementDropped(ctx, peer, question)
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

func (observers JudgementLedgerObservers) WatchFailed(ctx context.Context, err error) {
	for _, observer := range observers {
		observer.WatchFailed(ctx, err)
	}
}

func (observers JudgementLedgerObservers) JudgementUndecodable(
	ctx context.Context,
	key string,
	err error,
) {
	for _, observer := range observers {
		observer.JudgementUndecodable(ctx, key, err)
	}
}

func (observers JudgementLedgerObservers) WatchEnded(ctx context.Context) {
	for _, observer := range observers {
		observer.WatchEnded(ctx)
	}
}
