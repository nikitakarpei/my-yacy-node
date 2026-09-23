package replicaasks

import (
	"context"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type askKind[Ask any, Answered any] interface {
	askedFor() peerasks.AskedFor
	partitionKeyOf(ask Ask) partitionKey
	peerOf(ask Ask) yacymodel.Hash
	hedgeDelayOf(ctx context.Context, ask Ask) time.Duration
	putAsk(ctx context.Context, ask Ask) (Answered, bool)
	isCovering(answer Answered) bool
	amountOfDocumentsListedIn(answer Answered) int
}

type partitionKey struct {
	word      string
	partition uint
}

func askTheWordPartitions[Ask any, Answered any](
	ctx context.Context,
	asksInReplicaOrder []Ask,
	kind askKind[Ask, Answered],
	replicasCoveringAPartition int,
	observer ReplicaAsksObserver,
) peerasks.AskOutcomes[Ask, Answered] {
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

	return askOutcomesFrom(asksInReplicaOrder, answeredWordPartitions)
}

func wordPartitionsOf[Ask any, Answered any](
	asksInReplicaOrder []Ask,
	kind askKind[Ask, Answered],
	replicasCoveringAPartition int,
) []wordPartition[Ask, Answered] {
	wordPartitions := make([]wordPartition[Ask, Answered], 0, len(asksInReplicaOrder))
	peersAskedInTheRun := noAskedPeers()
	placeOfKey := make(map[partitionKey]int, len(asksInReplicaOrder))
	for placeInTheRun, ask := range asksInReplicaOrder {
		key := kind.partitionKeyOf(ask)
		place, known := placeOfKey[key]
		if !known {
			place = len(wordPartitions)
			placeOfKey[key] = place
			wordPartitions = append(wordPartitions, wordPartition[Ask, Answered]{
				kind:                       kind,
				replicasCoveringAPartition: replicasCoveringAPartition,
				askedPeers:                 peersAskedInTheRun,
			})
		}
		wordPartitions[place].asksInReplicaOrder = append(
			wordPartitions[place].asksInReplicaOrder, ask,
		)
		wordPartitions[place].placeOfEachAskInTheRun = append(
			wordPartitions[place].placeOfEachAskInTheRun, placeInTheRun,
		)
	}

	return wordPartitions
}

func settleTheWordPartitions[Ask any, Answered any](
	ctx context.Context,
	wordPartitions []wordPartition[Ask, Answered],
) []answeredWordPartition[Ask, Answered] {
	answeredWordPartitions := make([]answeredWordPartition[Ask, Answered], len(wordPartitions))
	unsettledWordPartitions, firstReplicasOfEachUnsettled := unsettledWordPartitionsWithTheirFirstReplicasOf(
		wordPartitions,
	)
	var settling sync.WaitGroup
	for place, unsettled := range unsettledWordPartitions {
		settling.Add(1)
		go func() {
			defer settling.Done()
			answeredWordPartitions[place] = unsettled.settle(
				ctx,
				firstReplicasOfEachUnsettled[place],
			)
		}()
	}
	settling.Wait()

	return answeredWordPartitions
}

func unsettledWordPartitionsWithTheirFirstReplicasOf[Ask any, Answered any](
	wordPartitions []wordPartition[Ask, Answered],
) ([]*unsettledWordPartition[Ask, Answered], [][]int) {
	unsettledWordPartitions := make(
		[]*unsettledWordPartition[Ask, Answered],
		0,
		len(wordPartitions),
	)
	firstReplicasOfEachUnsettled := make([][]int, 0, len(wordPartitions))
	for _, partition := range wordPartitions {
		unsettled := partition.unsettled()
		unsettledWordPartitions = append(unsettledWordPartitions, unsettled)
		firstReplicasOfEachUnsettled = append(
			firstReplicasOfEachUnsettled,
			unsettled.reserveTheFirstReplicas(),
		)
	}

	return unsettledWordPartitions, firstReplicasOfEachUnsettled
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

func askOutcomesFrom[Ask any, Answered any](
	asksInReplicaOrder []Ask,
	answeredWordPartitions []answeredWordPartition[Ask, Answered],
) peerasks.AskOutcomes[Ask, Answered] {
	askOutcomes := make(peerasks.AskOutcomes[Ask, Answered], 0, len(asksInReplicaOrder))
	for _, ask := range asksInReplicaOrder {
		askOutcomes = append(askOutcomes, peerasks.AskOutcome[Ask, Answered]{Ask: ask})
	}
	for _, answered := range answeredWordPartitions {
		for _, placed := range answered.placedAskOutcomes {
			askOutcomes[placed.placeInTheRun] = placed.outcome
		}
	}

	return askOutcomes
}
