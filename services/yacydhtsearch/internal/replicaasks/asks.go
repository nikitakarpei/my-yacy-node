// Package replicaasks puts the asks of a word partition to its replicas in
// turn, and settles the partition as soon as enough replicas have listed
// documents for the word.
package replicaasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
)

type PeerCalls interface {
	AskForMatchedAndHeldDocuments(
		ctx context.Context,
		asks []peerasks.MatchedAndHeldDocumentsAsk,
	) []peerasks.AnsweredMatchedAndHeldDocumentsAsk
	AskForMatchedDocuments(
		ctx context.Context,
		asks []peerasks.MatchedDocumentsAsk,
	) []peerasks.AnsweredMatchedDocumentsAsk
}

type HedgeDelay interface {
	HedgeDelayOf(ctx context.Context, peer peerdirectory.AskablePeer) time.Duration
}

type Asks struct {
	peerCalls                  PeerCalls
	hedgeDelay                 HedgeDelay
	replicasCoveringAPartition int
	observer                   ReplicaAsksObserver
}

func New(
	peerCalls PeerCalls,
	hedgeDelay HedgeDelay,
	replicasCoveringAPartition int,
	observer ReplicaAsksObserver,
) Asks {
	return Asks{
		peerCalls:                  peerCalls,
		hedgeDelay:                 hedgeDelay,
		replicasCoveringAPartition: replicasCoveringAPartition,
		observer:                   observer,
	}
}

func (asks Asks) AskForMatchedAndHeldDocuments(
	ctx context.Context,
	asksInReplicaOrder []peerasks.MatchedAndHeldDocumentsAsk,
) []peerasks.AnsweredMatchedAndHeldDocumentsAsk {
	return askTheWordPartitions(
		ctx,
		asksInReplicaOrder,
		matchedAndHeldDocumentsAskKind{peerCalls: asks.peerCalls, hedgeDelay: asks.hedgeDelay},
		asks.replicasCoveringAPartition,
		asks.observer,
	)
}

func (asks Asks) AskForMatchedDocuments(
	ctx context.Context,
	asksInReplicaOrder []peerasks.MatchedDocumentsAsk,
) []peerasks.AnsweredMatchedDocumentsAsk {
	return askTheWordPartitions(
		ctx,
		asksInReplicaOrder,
		matchedDocumentsAskKind{peerCalls: asks.peerCalls, hedgeDelay: asks.hedgeDelay},
		asks.replicasCoveringAPartition,
		asks.observer,
	)
}
