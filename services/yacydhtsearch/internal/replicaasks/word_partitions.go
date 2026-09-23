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

	wordPartitions := wordPartitionsOf(asksInReplicaOrder, kind, amountOfReplicasCoveringAPartition)
	askEachUntilSettled(askingContext, wordPartitions)

	observer.ReplicaAsksPerformed(
		ctx,
		performedReplicaAsksFrom(wordPartitions, time.Since(startedAt)),
	)

	return askOutcomesFrom(asksInReplicaOrder, wordPartitions)
}

func wordPartitionsOf[Ask any, Answered any](
	asksInReplicaOrder []Ask,
	kind askKind[Ask, Answered],
	amountOfReplicasCoveringAPartition int,
) []*wordPartition[Ask, Answered] {
	chosenPeers := noChosenPeers()
	asksOfEachWordPartition := asksOfEachWordPartitionIn(asksInReplicaOrder, kind)
	wordPartitions := make([]*wordPartition[Ask, Answered], 0, len(asksOfEachWordPartition))
	for _, asksOfTheWordPartition := range asksOfEachWordPartition {
		wordPartitions = append(wordPartitions, wordPartitionOf(
			asksOfTheWordPartition,
			kind,
			amountOfReplicasCoveringAPartition,
			chosenPeers,
		))
	}

	return wordPartitions
}

func asksOfEachWordPartitionIn[Ask any, Answered any](
	asksInReplicaOrder []Ask,
	kind askKind[Ask, Answered],
) [][]placedAsk[Ask] {
	asksOfEachWordPartition := make([][]placedAsk[Ask], 0, len(asksInReplicaOrder))
	placeOfKey := make(map[wordPartitionKey]int, len(asksInReplicaOrder))
	for placeInTheRun, ask := range asksInReplicaOrder {
		key := kind.wordPartitionKeyOf(ask)
		place, known := placeOfKey[key]
		if !known {
			place = len(asksOfEachWordPartition)
			placeOfKey[key] = place
			asksOfEachWordPartition = append(asksOfEachWordPartition, nil)
		}
		asksOfEachWordPartition[place] = append(
			asksOfEachWordPartition[place],
			placedAsk[Ask]{ask: ask, placeInTheRun: placeInTheRun},
		)
	}

	return asksOfEachWordPartition
}

func askEachUntilSettled[Ask any, Answered any](
	ctx context.Context,
	wordPartitions []*wordPartition[Ask, Answered],
) {
	for _, partition := range wordPartitions {
		partition.chooseTheFirstAsks()
	}
	var settling sync.WaitGroup
	for _, partition := range wordPartitions {
		settling.Add(1)
		go func() {
			defer settling.Done()
			partition.askUntilSettled(ctx)
		}()
	}
	settling.Wait()
}

func performedReplicaAsksFrom[Ask any, Answered any](
	wordPartitions []*wordPartition[Ask, Answered],
	timeSpent time.Duration,
) PerformedReplicaAsks {
	settledWordPartitions := make([]SettledWordPartition, 0, len(wordPartitions))
	for _, partition := range wordPartitions {
		settledWordPartitions = append(settledWordPartitions, partition.settledWordPartition())
	}

	return PerformedReplicaAsks{
		EndedBy:        endedByOf(wordPartitions),
		TimeSpent:      timeSpent,
		WordPartitions: settledWordPartitions,
	}
}

func endedByOf[Ask any, Answered any](
	wordPartitions []*wordPartition[Ask, Answered],
) EndedBy {
	for _, partition := range wordPartitions {
		if partition.settledBy == SettledByDeadline {
			return EndedByDeadline
		}
	}

	return EndedByCoverage
}

func askOutcomesFrom[Ask any, Answered any](
	asksInReplicaOrder []Ask,
	wordPartitions []*wordPartition[Ask, Answered],
) peerasks.AskOutcomes[Ask, Answered] {
	askOutcomes := make(peerasks.AskOutcomes[Ask, Answered], 0, len(asksInReplicaOrder))
	for _, ask := range asksInReplicaOrder {
		askOutcomes = append(askOutcomes, peerasks.AskOutcome[Ask, Answered]{Ask: ask})
	}
	for _, partition := range wordPartitions {
		for _, placed := range partition.placedAskOutcomes() {
			askOutcomes[placed.placeInTheRun] = placed.outcome
		}
	}

	return askOutcomes
}
