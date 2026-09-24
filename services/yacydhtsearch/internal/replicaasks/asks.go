// Package replicaasks runs the asks of one query as one run. The consumer
// sends asks to the run at any time and closes the asks when it has no more.
// The run puts the asks of each word partition to its replicas in turn,
// settles the partition as soon as enough replicas have searched for the word
// or listed documents for it, and sends each settled word partition with the
// outcome of each of its asks. The consumer reads the settled word partitions until they close.
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
}

type HedgeDelay interface {
	HedgeDelayOf(ctx context.Context, peer peerdirectory.AskablePeer) time.Duration
}

type Asks struct {
	peerCalls                          PeerCalls
	hedgeDelay                         HedgeDelay
	amountOfReplicasCoveringAPartition int
	observer                           ReplicaAsksObserver
}

func New(
	peerCalls PeerCalls,
	hedgeDelay HedgeDelay,
	amountOfReplicasCoveringAPartition int,
	observer ReplicaAsksObserver,
) Asks {
	return Asks{
		peerCalls:                          peerCalls,
		hedgeDelay:                         hedgeDelay,
		amountOfReplicasCoveringAPartition: amountOfReplicasCoveringAPartition,
		observer:                           observer,
	}
}

type Run struct {
	Asks                  chan<- []peerasks.SearchDocumentsAsk
	SettledWordPartitions <-chan SettledWordPartition
}

func (replicaAsks Asks) Start(ctx context.Context) Run {
	asks := make(chan []peerasks.SearchDocumentsAsk)
	settledWordPartitions := make(chan SettledWordPartition)
	go openRunOf(replicaAsks, asks, settledWordPartitions).askUntilOver(ctx)

	return Run{Asks: asks, SettledWordPartitions: settledWordPartitions}
}
