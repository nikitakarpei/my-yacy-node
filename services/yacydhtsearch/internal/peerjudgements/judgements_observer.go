package peerjudgements

import "context"

type JudgementsObserver interface {
	PeerStood(ctx context.Context, form Form, standing PeerStanding)
	PeerJudged(ctx context.Context, form Form, judgedPeer JudgedPeer)
}

type JudgementsObservers []JudgementsObserver

func (observers JudgementsObservers) PeerStood(
	ctx context.Context,
	form Form,
	standing PeerStanding,
) {
	for _, observer := range observers {
		observer.PeerStood(ctx, form, standing)
	}
}

func (observers JudgementsObservers) PeerJudged(
	ctx context.Context,
	form Form,
	judgedPeer JudgedPeer,
) {
	for _, observer := range observers {
		observer.PeerJudged(ctx, form, judgedPeer)
	}
}
