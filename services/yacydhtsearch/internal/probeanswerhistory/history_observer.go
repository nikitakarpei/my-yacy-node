package probeanswerhistory

import "context"

type HistoryObserver interface {
	ProbeAnswerAppendFailed(ctx context.Context, err error)
	ProbeAnswerUnreadable(ctx context.Context, position ProbeAnswerPosition, err error)
	ProbeAnswerHistoryEnded(ctx context.Context, err error)
}

type HistoryObservers []HistoryObserver

func (observers HistoryObservers) ProbeAnswerAppendFailed(ctx context.Context, err error) {
	for _, observer := range observers {
		observer.ProbeAnswerAppendFailed(ctx, err)
	}
}

func (observers HistoryObservers) ProbeAnswerUnreadable(
	ctx context.Context,
	position ProbeAnswerPosition,
	err error,
) {
	for _, observer := range observers {
		observer.ProbeAnswerUnreadable(ctx, position, err)
	}
}

func (observers HistoryObservers) ProbeAnswerHistoryEnded(ctx context.Context, err error) {
	for _, observer := range observers {
		observer.ProbeAnswerHistoryEnded(ctx, err)
	}
}
