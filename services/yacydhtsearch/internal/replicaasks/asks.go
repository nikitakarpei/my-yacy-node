// Package replicaasks puts the asks of a partition to its replicas in turn,
// settles the partition as soon as enough answers of the replicas cover it,
// and reports for each ask whether it was put and what the peer answered.
package replicaasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
)

type PeerCalls interface {
	AskForSearchDocuments(
		ctx context.Context,
		asks []peerasks.SearchDocumentsAsk,
	) []peerasks.AnsweredSearchDocumentsAsk
	AskForURLMetadata(
		ctx context.Context,
		asks []peerasks.URLMetadataAsk,
	) []peerasks.AnsweredURLMetadataAsk
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

func (asks Asks) AskForSearchDocuments(
	ctx context.Context,
	asksInReplicaOrder []peerasks.SearchDocumentsAsk,
) peerasks.SearchDocumentsAskOutcomes {
	return askTheWordPartitions(
		ctx,
		asksInReplicaOrder,
		searchDocumentsAskKind{peerCalls: asks.peerCalls, hedgeDelay: asks.hedgeDelay},
		asks.replicasCoveringAPartition,
		asks.observer,
	)
}

const replicasCoveringTheURLMetadataOfAPartition = 1

func (asks Asks) AskForURLMetadata(
	ctx context.Context,
	asksInReplicaOrder []peerasks.URLMetadataAsk,
) peerasks.URLMetadataAskOutcomes {
	return askTheWordPartitions(
		ctx,
		asksInReplicaOrder,
		urlMetadataAskKind{peerCalls: asks.peerCalls, hedgeDelay: asks.hedgeDelay},
		replicasCoveringTheURLMetadataOfAPartition,
		asks.observer,
	)
}
