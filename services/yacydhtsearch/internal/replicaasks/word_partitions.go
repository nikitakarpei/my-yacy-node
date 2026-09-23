package replicaasks

import (
	"context"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type askKind[Ask any, Answered any] interface {
	askedFor() peerasks.AskedFor
	wordPartitionKeyOf(ask Ask) wordPartitionKey
	hedgeDelayOf(ctx context.Context, ask Ask) time.Duration
	putAsk(ctx context.Context, ask Ask) (Answered, bool)
	isCovering(answer Answered) bool
	amountOfDocumentsListedIn(answer Answered) int
}

type wordPartitionKey struct {
	word      string
	partition uint
}

func askTheWordPartitions[Ask any, Answered any](
	ctx context.Context,
	asksInReplicaOrder []Ask,
	kind askKind[Ask, Answered],
	replicasCoveringAPartition int,
	observer ReplicaAsksObserver,
) peerasks.AsksPut[Ask, Answered] {
	startedAt := time.Now()
	askingContext, stopAsking := context.WithCancel(ctx)
	defer stopAsking()

	answeredWordPartitions := settleTheWordPartitions(
		askingContext,
		wordPartitionsOf(asksInReplicaOrder, kind, replicasCoveringAPartition),
	)

	observer.ReplicaAsksPerformed(
		ctx,
		performedReplicaAsksFrom(kind.askedFor(), answeredWordPartitions, time.Since(startedAt)),
	)

	return asksPutFrom(answeredWordPartitions)
}

func wordPartitionsOf[Ask any, Answered any](
	asksInReplicaOrder []Ask,
	kind askKind[Ask, Answered],
	replicasCoveringAPartition int,
) []wordPartition[Ask, Answered] {
	wordPartitions := make([]wordPartition[Ask, Answered], 0, len(asksInReplicaOrder))
	placeOfKey := make(map[wordPartitionKey]int, len(asksInReplicaOrder))
	for _, ask := range asksInReplicaOrder {
		key := kind.wordPartitionKeyOf(ask)
		place, known := placeOfKey[key]
		if !known {
			place = len(wordPartitions)
			placeOfKey[key] = place
			wordPartitions = append(wordPartitions, wordPartition[Ask, Answered]{
				kind:                       kind,
				replicasCoveringAPartition: replicasCoveringAPartition,
			})
		}
		wordPartitions[place].asksInReplicaOrder = append(
			wordPartitions[place].asksInReplicaOrder, ask,
		)
	}

	return wordPartitions
}

func settleTheWordPartitions[Ask any, Answered any](
	ctx context.Context,
	wordPartitions []wordPartition[Ask, Answered],
) []answeredWordPartition[Ask, Answered] {
	answeredWordPartitions := make([]answeredWordPartition[Ask, Answered], len(wordPartitions))
	var settling sync.WaitGroup
	for place, partition := range wordPartitions {
		settling.Add(1)
		go func() {
			defer settling.Done()
			answeredWordPartitions[place] = partition.settle(ctx)
		}()
	}
	settling.Wait()

	return answeredWordPartitions
}

func performedReplicaAsksFrom[Ask any, Answered any](
	askedFor peerasks.AskedFor,
	answeredWordPartitions []answeredWordPartition[Ask, Answered],
	timeSpent time.Duration,
) PerformedReplicaAsks {
	settledWordPartitions := make([]SettledWordPartition, 0, len(answeredWordPartitions))
	for _, answered := range answeredWordPartitions {
		settledWordPartitions = append(settledWordPartitions, answered.settled)
	}

	return PerformedReplicaAsks{
		AskedFor:       askedFor,
		EndedBy:        endedByOf(answeredWordPartitions),
		TimeSpent:      timeSpent,
		WordPartitions: settledWordPartitions,
	}
}

func endedByOf[Ask any, Answered any](
	answeredWordPartitions []answeredWordPartition[Ask, Answered],
) EndedBy {
	for _, answered := range answeredWordPartitions {
		if answered.settled.SettledBy == SettledByDeadline {
			return EndedByDeadline
		}
	}

	return EndedByCoverage
}

func asksPutFrom[Ask any, Answered any](
	answeredWordPartitions []answeredWordPartition[Ask, Answered],
) peerasks.AsksPut[Ask, Answered] {
	asksPut := peerasks.AsksPut[Ask, Answered]{
		Asks:         make([]Ask, 0, len(answeredWordPartitions)),
		AnsweredAsks: make([]Answered, 0, len(answeredWordPartitions)),
	}
	for _, answered := range answeredWordPartitions {
		asksPut.Asks = append(asksPut.Asks, answered.asksPut...)
		asksPut.AnsweredAsks = append(asksPut.AnsweredAsks, answered.answers...)
	}

	return asksPut
}
