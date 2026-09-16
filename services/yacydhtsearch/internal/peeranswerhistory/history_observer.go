package peeranswerhistory

import "context"

type HistoryObserver interface {
	PeerAnswerAppendFailed(ctx context.Context, err error)
	PeerAnswerUnreadable(ctx context.Context, position AnswerPosition, err error)
	PeerAnswerHistoryEnded(ctx context.Context, err error)
}

type HistoryObservers []HistoryObserver

func (observers HistoryObservers) PeerAnswerAppendFailed(ctx context.Context, err error) {
	for _, observer := range observers {
		observer.PeerAnswerAppendFailed(ctx, err)
	}
}

func (observers HistoryObservers) PeerAnswerUnreadable(
	ctx context.Context,
	position AnswerPosition,
	err error,
) {
	for _, observer := range observers {
		observer.PeerAnswerUnreadable(ctx, position, err)
	}
}

func (observers HistoryObservers) PeerAnswerHistoryEnded(ctx context.Context, err error) {
	for _, observer := range observers {
		observer.PeerAnswerHistoryEnded(ctx, err)
	}
}
