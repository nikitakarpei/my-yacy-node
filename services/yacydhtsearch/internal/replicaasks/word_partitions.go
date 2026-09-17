package replicaasks

import (
	"context"
	"sync"
	"time"
)

type replicaCalls[Ask any, Answered any] struct {
	wordPartitionKeyOf        func(Ask) wordPartitionKey
	hedgeDelayOf              func(context.Context, Ask) time.Duration
	putAsk                    func(context.Context, Ask) (Answered, bool)
	amountOfDocumentsListedIn func(Answered) int
}

type wordPartitionKey struct {
	word      string
	partition uint
}

func answersOfWordPartitions[Ask any, Answered any](
	ctx context.Context,
	asksInReplicaOrder []Ask,
	calls replicaCalls[Ask, Answered],
	replicasCoveringAPartition int,
	observer ReplicaAsksObserver,
) []Answered {
	startedAt := time.Now()
	askingContext, stopAsking := context.WithCancel(ctx)
	defer stopAsking()

	settledWordPartitions := settledWordPartitionsFrom(
		askingContext,
		wordPartitionsOf(asksInReplicaOrder, calls, replicasCoveringAPartition),
	)

	observer.ReplicaAsksPerformed(
		ctx,
		performedReplicaAsksFrom(settledWordPartitions, time.Since(startedAt)),
	)

	return answersOf(settledWordPartitions)
}

func wordPartitionsOf[Ask any, Answered any](
	asksInReplicaOrder []Ask,
	calls replicaCalls[Ask, Answered],
	replicasCoveringAPartition int,
) []wordPartition[Ask, Answered] {
	wordPartitions := make([]wordPartition[Ask, Answered], 0, len(asksInReplicaOrder))
	placeOfKey := make(map[wordPartitionKey]int, len(asksInReplicaOrder))
	for _, ask := range asksInReplicaOrder {
		key := calls.wordPartitionKeyOf(ask)
		place, known := placeOfKey[key]
		if !known {
			placeOfKey[key] = len(wordPartitions)
			wordPartitions = append(wordPartitions, wordPartition[Ask, Answered]{
				calls:                      calls,
				replicasCoveringAPartition: replicasCoveringAPartition,
			})
			place = placeOfKey[key]
		}
		wordPartitions[place].replicas = append(wordPartitions[place].replicas, ask)
	}

	return wordPartitions
}

func settledWordPartitionsFrom[Ask any, Answered any](
	ctx context.Context,
	wordPartitions []wordPartition[Ask, Answered],
) []settledWordPartition[Answered] {
	settledWordPartitions := make([]settledWordPartition[Answered], len(wordPartitions))
	var asking sync.WaitGroup
	for place, partition := range wordPartitions {
		asking.Add(1)
		go func() {
			defer asking.Done()
			settledWordPartitions[place] = partition.settle(ctx)
		}()
	}
	asking.Wait()

	return settledWordPartitions
}

func performedReplicaAsksFrom[Answered any](
	settledWordPartitions []settledWordPartition[Answered],
	timeSpent time.Duration,
) PerformedReplicaAsks {
	performedWordPartitions := make([]PerformedWordPartition, 0, len(settledWordPartitions))
	for _, settled := range settledWordPartitions {
		performedWordPartitions = append(performedWordPartitions, PerformedWordPartition{
			SettledBy:               settled.settledBy,
			CoveringAskPutOn:        settled.coveringAskPutOn,
			AmountOfDocumentsListed: settled.amountOfDocumentsListed,
			Asks:                    performedReplicaAsksOf(settled),
		})
	}

	return PerformedReplicaAsks{
		EndedBy:        endedByOf(settledWordPartitions),
		TimeSpent:      timeSpent,
		WordPartitions: performedWordPartitions,
	}
}

func performedReplicaAsksOf[Answered any](
	settled settledWordPartition[Answered],
) []PerformedReplicaAsk {
	performedReplicaAsks := make([]PerformedReplicaAsk, 0, len(settled.putOn))
	for _, putOn := range settled.putOn {
		performedReplicaAsks = append(performedReplicaAsks, PerformedReplicaAsk{PutOn: putOn})
	}

	return performedReplicaAsks
}

func endedByOf[Answered any](settledWordPartitions []settledWordPartition[Answered]) EndedBy {
	for _, settled := range settledWordPartitions {
		if settled.settledBy == SettledByDeadline {
			return EndedByDeadline
		}
	}

	return EndedByCoverage
}

func answersOf[Answered any](
	settledWordPartitions []settledWordPartition[Answered],
) []Answered {
	answers := make([]Answered, 0, len(settledWordPartitions))
	for _, settled := range settledWordPartitions {
		answers = append(answers, settled.answers...)
	}

	return answers
}
