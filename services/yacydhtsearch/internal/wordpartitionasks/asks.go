// Package wordpartitionasks runs the asks of one query as one run. The consumer
// sends asks to the run at any time and closes the asks when it has no more.
// The run puts the ask of each word partition to its replicas in turn,
// settles the ask as soon as enough replicas have searched for the word
// or listed documents for it, and sends each settled ask with the answers of
// its replicas. The consumer reads the settled asks until they close.
package wordpartitionasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
)

type ReplicaCalls interface {
	Put(ctx context.Context, ask Ask, replica peerdirectory.AskablePeer) (ReplicaAnswer, bool)
}

type HedgeDelay interface {
	HedgeDelayOf(ctx context.Context, peer peerdirectory.AskablePeer) time.Duration
}

type Asks struct {
	replicaCalls                       ReplicaCalls
	hedgeDelay                         HedgeDelay
	clock                              Clock
	amountOfReplicasCoveringAPartition int
	observer                           WordPartitionAsksObserver
}

func New(
	replicaCalls ReplicaCalls,
	hedgeDelay HedgeDelay,
	clock Clock,
	amountOfReplicasCoveringAPartition int,
	observer WordPartitionAsksObserver,
) Asks {
	return Asks{
		replicaCalls:                       replicaCalls,
		hedgeDelay:                         hedgeDelay,
		clock:                              clock,
		amountOfReplicasCoveringAPartition: amountOfReplicasCoveringAPartition,
		observer:                           observer,
	}
}

type Run struct {
	Asks        chan<- []Ask
	SettledAsks <-chan SettledAsk
}

func (asks Asks) Start(ctx context.Context) Run {
	sentAsks := make(chan []Ask)
	settledAsks := make(chan SettledAsk)
	go openRunOf(asks, sentAsks, settledAsks).askUntilOver(ctx)

	return Run{Asks: sentAsks, SettledAsks: settledAsks}
}

func (asks Asks) startTheHedgeTimer(
	ctx context.Context,
	peer peerdirectory.AskablePeer,
	hedgeDue func(),
) (stop func()) {
	return asks.clock.After(asks.hedgeDelay.HedgeDelayOf(ctx, peer), hedgeDue)
}

func (asks Asks) askTheReplica(
	ctx context.Context,
	ask Ask,
	replica peerdirectory.AskablePeer,
) (ReplicaAnswer, bool) {
	return asks.replicaCalls.Put(ctx, ask, replica)
}

func (asks Asks) reportPerformed(ctx context.Context, performed PerformedWordPartitionAsks) {
	asks.observer.WordPartitionAsksPerformed(ctx, performed)
}
