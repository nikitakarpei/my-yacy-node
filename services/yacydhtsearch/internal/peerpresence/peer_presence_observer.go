package peerpresence

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerPresenceObserver interface {
	PeerAnsweredForTheFirstTime(ctx context.Context, peer yacymodel.Hash, address string)
	PeerEarnedPresence(ctx context.Context, peer yacymodel.Hash, presence time.Duration)
	PeersObserved(ctx context.Context, amountOfObservedPeers int)
}

type PeerPresenceObservers []PeerPresenceObserver

func (observers PeerPresenceObservers) PeerAnsweredForTheFirstTime(
	ctx context.Context,
	peer yacymodel.Hash,
	address string,
) {
	for _, observer := range observers {
		observer.PeerAnsweredForTheFirstTime(ctx, peer, address)
	}
}

func (observers PeerPresenceObservers) PeerEarnedPresence(
	ctx context.Context,
	peer yacymodel.Hash,
	presence time.Duration,
) {
	for _, observer := range observers {
		observer.PeerEarnedPresence(ctx, peer, presence)
	}
}

func (observers PeerPresenceObservers) PeersObserved(
	ctx context.Context,
	amountOfObservedPeers int,
) {
	for _, observer := range observers {
		observer.PeersObserved(ctx, amountOfObservedPeers)
	}
}
