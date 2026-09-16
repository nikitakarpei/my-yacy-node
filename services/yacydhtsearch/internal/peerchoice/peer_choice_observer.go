package peerchoice

import "context"

type PeerChoiceObserver interface {
	PeersSelected(ctx context.Context, ringFractions []float64)
}

type PeerChoiceObservers []PeerChoiceObserver

func (observers PeerChoiceObservers) PeersSelected(
	ctx context.Context,
	ringFractions []float64,
) {
	for _, observer := range observers {
		observer.PeersSelected(ctx, ringFractions)
	}
}
