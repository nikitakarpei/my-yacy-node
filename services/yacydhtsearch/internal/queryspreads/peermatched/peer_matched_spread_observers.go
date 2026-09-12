package peermatched

import "context"

type PeerMatchedSpreadObserver interface {
	PeerMatchedSpreadPerformed(ctx context.Context, spread PerformedPeerMatchedSpread)
}

type PeerMatchedSpreadObservers []PeerMatchedSpreadObserver

func (observers PeerMatchedSpreadObservers) PeerMatchedSpreadPerformed(
	ctx context.Context,
	spread PerformedPeerMatchedSpread,
) {
	for _, observer := range observers {
		observer.PeerMatchedSpreadPerformed(ctx, spread)
	}
}
