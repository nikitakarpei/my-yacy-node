package peerchoice

import "context"

type PeerChoiceObserver interface {
	PeersTakenFromTheRing(ctx context.Context, ringFractions []float64)
}

type PeerChoiceObservers []PeerChoiceObserver

func (observers PeerChoiceObservers) PeersTakenFromTheRing(
	ctx context.Context,
	ringFractions []float64,
) {
	for _, observer := range observers {
		observer.PeersTakenFromTheRing(ctx, ringFractions)
	}
}
