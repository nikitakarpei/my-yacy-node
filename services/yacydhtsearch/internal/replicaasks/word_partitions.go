package replicaasks

import (
	"context"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type askKind[Ask any, Answered any] interface {
	wordPartitionKeyOf(ask Ask) wordPartitionKey
	peerOf(ask Ask) yacymodel.Hash
	hedgeDelayOf(ctx context.Context, ask Ask) time.Duration
	putAsk(ctx context.Context, ask Ask) (Answered, bool)
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
	amountOfReplicasCoveringAPartition int,
	observer ReplicaAsksObserver,
) peerasks.AskOutcomes[Ask, Answered] {
	startedAt := time.Now()
	askingContext, stopAsking := context.WithCancel(ctx)
	defer stopAsking()

	askedWordPartitions := settleTheWordPartitions(
		askingContext,
		wordPartitionsOf(asksInReplicaOrder, kind, amountOfReplicasCoveringAPartition),
	)

	observer.ReplicaAsksPerformed(
		ctx,
		performedReplicaAsksFrom(askedWordPartitions, time.Since(startedAt)),
	)

	return askOutcomesFrom(asksInReplicaOrder, askedWordPartitions)
}

func wordPartitionsOf[Ask any, Answered any](
	asksInReplicaOrder []Ask,
	kind askKind[Ask, Answered],
	amountOfReplicasCoveringAPartition int,
) []wordPartition[Ask, Answered] {
	wordPartitions := make([]wordPartition[Ask, Answered], 0, len(asksInReplicaOrder))
	reservedPeers := noReservedPeers()
	placeOfKey := make(map[wordPartitionKey]int, len(asksInReplicaOrder))
	for placeInTheRun, ask := range asksInReplicaOrder {
		key := kind.wordPartitionKeyOf(ask)
		place, known := placeOfKey[key]
		if !known {
			place = len(wordPartitions)
			placeOfKey[key] = place
			wordPartitions = append(wordPartitions, wordPartition[Ask, Answered]{
				kind:                               kind,
				amountOfReplicasCoveringAPartition: amountOfReplicasCoveringAPartition,
				reservedPeers:                      reservedPeers,
			})
		}
		wordPartitions[place].asksInReplicaOrder = append(
			wordPartitions[place].asksInReplicaOrder,
			placedAsk[Ask]{ask: ask, placeInTheRun: placeInTheRun},
		)
	}

	return wordPartitions
}

func settleTheWordPartitions[Ask any, Answered any](
	ctx context.Context,
	wordPartitions []wordPartition[Ask, Answered],
) []askedWordPartition[Ask, Answered] {
	askings := askingsOf(wordPartitions)
	for _, asking := range askings {
		asking.reserveTheFirstAsks()
	}
	askedWordPartitions := make([]askedWordPartition[Ask, Answered], len(askings))
	var settling sync.WaitGroup
	for place, asking := range askings {
		settling.Add(1)
		go func() {
			defer settling.Done()
			askedWordPartitions[place] = asking.settle(ctx)
		}()
	}
	settling.Wait()

	return askedWordPartitions
}

func askingsOf[Ask any, Answered any](
	wordPartitions []wordPartition[Ask, Answered],
) []*wordPartitionAsking[Ask, Answered] {
	askings := make([]*wordPartitionAsking[Ask, Answered], 0, len(wordPartitions))
	for _, partition := range wordPartitions {
		askings = append(askings, partition.asking())
	}

	return askings
}

func performedReplicaAsksFrom[Ask any, Answered any](
	askedWordPartitions []askedWordPartition[Ask, Answered],
	timeSpent time.Duration,
) PerformedReplicaAsks {
	settledWordPartitions := make([]SettledWordPartition, 0, len(askedWordPartitions))
	for _, asked := range askedWordPartitions {
		settledWordPartitions = append(settledWordPartitions, asked.settledWordPartition)
	}

	return PerformedReplicaAsks{
		EndedBy:        endedByOf(askedWordPartitions),
		TimeSpent:      timeSpent,
		WordPartitions: settledWordPartitions,
	}
}

func endedByOf[Ask any, Answered any](
	askedWordPartitions []askedWordPartition[Ask, Answered],
) EndedBy {
	for _, asked := range askedWordPartitions {
		if asked.settledWordPartition.SettledBy == SettledByDeadline {
			return EndedByDeadline
		}
	}

	return EndedByCoverage
}

func askOutcomesFrom[Ask any, Answered any](
	asksInReplicaOrder []Ask,
	askedWordPartitions []askedWordPartition[Ask, Answered],
) peerasks.AskOutcomes[Ask, Answered] {
	askOutcomes := make(peerasks.AskOutcomes[Ask, Answered], 0, len(asksInReplicaOrder))
	for _, ask := range asksInReplicaOrder {
		askOutcomes = append(askOutcomes, peerasks.AskOutcome[Ask, Answered]{Ask: ask})
	}
	for _, asked := range askedWordPartitions {
		for _, placed := range asked.placedAskOutcomes {
			askOutcomes[placed.placeInTheRun] = placed.outcome
		}
	}

	return askOutcomes
}
