package presenceaccrual

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PresenceAccrualObserver interface {
	PeerAnsweredForTheFirstTime(ctx context.Context, peer yacymodel.Hash, address string)
	PeerEarnedPresence(ctx context.Context, peer yacymodel.Hash, presence time.Duration)
	PeersObserved(ctx context.Context, amountOfObservedPeers int)
}

type PresenceAccrualObservers []PresenceAccrualObserver

func (observers PresenceAccrualObservers) PeerAnsweredForTheFirstTime(
	ctx context.Context,
	peer yacymodel.Hash,
	address string,
) {
	for _, observer := range observers {
		observer.PeerAnsweredForTheFirstTime(ctx, peer, address)
	}
}

func (observers PresenceAccrualObservers) PeerEarnedPresence(
	ctx context.Context,
	peer yacymodel.Hash,
	presence time.Duration,
) {
	for _, observer := range observers {
		observer.PeerEarnedPresence(ctx, peer, presence)
	}
}

func (observers PresenceAccrualObservers) PeersObserved(
	ctx context.Context,
	amountOfObservedPeers int,
) {
	for _, observer := range observers {
		observer.PeersObserved(ctx, amountOfObservedPeers)
	}
}
