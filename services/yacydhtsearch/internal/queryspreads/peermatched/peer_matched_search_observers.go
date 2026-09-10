package peermatched

import "context"

type PeerMatchedSearchObserver interface {
	PeerMatchedSearchPerformed(ctx context.Context, search PerformedPeerMatchedSearch)
}

type PeerMatchedSearchObservers []PeerMatchedSearchObserver

func (observers PeerMatchedSearchObservers) PeerMatchedSearchPerformed(
	ctx context.Context,
	search PerformedPeerMatchedSearch,
) {
	for _, observer := range observers {
		observer.PeerMatchedSearchPerformed(ctx, search)
	}
}
