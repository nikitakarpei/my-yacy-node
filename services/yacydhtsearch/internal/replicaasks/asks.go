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

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
)

type ReplicaCalls[Ask any, Answered any] interface {
	ReplicaOf(ask Ask) Replica
	AnswerTo(ctx context.Context, ask Ask) (Answered, bool)
	CoverageFrom(answer Answered) Coverage
}

type HedgeDelay interface {
	HedgeDelayOf(ctx context.Context, peer peerdirectory.AskablePeer) time.Duration
}

type Asks[Ask any, Answered any] struct {
	replicaCalls                       ReplicaCalls[Ask, Answered]
	hedgeDelay                         HedgeDelay
	amountOfReplicasCoveringAPartition int
	observer                           ReplicaAsksObserver
}

func New[Ask any, Answered any](
	replicaCalls ReplicaCalls[Ask, Answered],
	hedgeDelay HedgeDelay,
	amountOfReplicasCoveringAPartition int,
	observer ReplicaAsksObserver,
) Asks[Ask, Answered] {
	return Asks[Ask, Answered]{
		replicaCalls:                       replicaCalls,
		hedgeDelay:                         hedgeDelay,
		amountOfReplicasCoveringAPartition: amountOfReplicasCoveringAPartition,
		observer:                           observer,
	}
}

type Run[Ask any, Answered any] struct {
	Asks                  chan<- []Ask
	SettledWordPartitions <-chan SettledWordPartition[Ask, Answered]
}

func (replicaAsks Asks[Ask, Answered]) Start(ctx context.Context) Run[Ask, Answered] {
	asks := make(chan []Ask)
	settledWordPartitions := make(chan SettledWordPartition[Ask, Answered])
	go openRunOf(replicaAsks, asks, settledWordPartitions).askUntilOver(ctx)

	return Run[Ask, Answered]{Asks: asks, SettledWordPartitions: settledWordPartitions}
}
