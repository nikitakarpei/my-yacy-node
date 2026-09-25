package replicaasks

import (
	"context"
	"time"
)

type openRun[Ask any, Answered any] struct {
	replicaAsks                     Asks[Ask, Answered]
	startedAt                       time.Time
	asks                            <-chan []Ask
	settledWordPartitions           chan<- SettledWordPartition[Ask, Answered]
	askedWordPartitions             chan *wordPartition[Ask, Answered]
	chosenPeers                     *chosenPeers
	wordPartitions                  []*wordPartition[Ask, Answered]
	wordPartitionKeys               map[wordPartitionKey]struct{}
	amountOfWordPartitionsUnsettled int
	settledWordPartitionsUnread     []SettledWordPartition[Ask, Answered]
}

type wordPartitionKey struct {
	word      string
	partition uint
}

func openRunOf[Ask any, Answered any](
	replicaAsks Asks[Ask, Answered],
	asks <-chan []Ask,
	settledWordPartitions chan<- SettledWordPartition[Ask, Answered],
) *openRun[Ask, Answered] {
	return &openRun[Ask, Answered]{
		replicaAsks:           replicaAsks,
		startedAt:             time.Now(),
		asks:                  asks,
		settledWordPartitions: settledWordPartitions,
		askedWordPartitions:   make(chan *wordPartition[Ask, Answered]),
		chosenPeers:           noChosenPeers(),
		wordPartitionKeys:     map[wordPartitionKey]struct{}{},
	}
}

func (run *openRun[Ask, Answered]) askUntilOver(ctx context.Context) {
	for !run.isOver() {
		run.takeTheNextEvent(ctx)
	}
	run.replicaAsks.observer.ReplicaAsksPerformed(ctx, run.performedReplicaAsks())
	close(run.settledWordPartitions)
}

func (run *openRun[Ask, Answered]) isOver() bool {
	return run.asks == nil && run.amountOfWordPartitionsUnsettled == 0 &&
		len(run.settledWordPartitionsUnread) == 0
}

func (run *openRun[Ask, Answered]) takeTheNextEvent(ctx context.Context) {
	select {
	case asks, open := <-run.asks:
		run.takeTheAsks(ctx, asks, open)
	case partition := <-run.askedWordPartitions:
		run.takeTheSettledWordPartition(partition)
	case run.readerOfTheNextSettledWordPartition() <- run.nextSettledWordPartition():
		run.settledWordPartitionsUnread = run.settledWordPartitionsUnread[1:]
	}
}

func (run *openRun[Ask, Answered]) takeTheAsks(
	ctx context.Context,
	asksInReplicaOrder []Ask,
	open bool,
) {
	if !open {
		run.asks = nil

		return
	}
	addedWordPartitions := run.wordPartitionsOf(asksInReplicaOrder)
	for _, partition := range addedWordPartitions {
		partition.chooseTheFirstAsks()
	}
	for _, partition := range addedWordPartitions {
		run.startAsking(ctx, partition)
	}
}

func (run *openRun[Ask, Answered]) wordPartitionsOf(
	asksInReplicaOrder []Ask,
) []*wordPartition[Ask, Answered] {
	asksOfEachWordPartition := run.asksOfEachAddedWordPartitionIn(asksInReplicaOrder)
	addedWordPartitions := make([]*wordPartition[Ask, Answered], 0, len(asksOfEachWordPartition))
	for _, asksOfTheWordPartition := range asksOfEachWordPartition {
		addedWordPartitions = append(addedWordPartitions, wordPartitionOf(
			asksOfTheWordPartition,
			run.replicaAsks,
			run.chosenPeers,
		))
	}
	run.wordPartitions = append(run.wordPartitions, addedWordPartitions...)

	return addedWordPartitions
}

func (run *openRun[Ask, Answered]) asksOfEachAddedWordPartitionIn(
	asksInReplicaOrder []Ask,
) [][]placedAsk[Ask] {
	asksOfEachWordPartition := make([][]placedAsk[Ask], 0, len(asksInReplicaOrder))
	indexOfEachAddedWordPartition := make(map[wordPartitionKey]int, len(asksInReplicaOrder))
	for _, ask := range asksInReplicaOrder {
		replica := run.replicaAsks.replicaCalls.ReplicaOf(ask)
		key := wordPartitionKeyOf(replica)
		index, added := indexOfEachAddedWordPartition[key]
		if !added {
			if _, alreadyInTheRun := run.wordPartitionKeys[key]; alreadyInTheRun {
				continue
			}
			run.wordPartitionKeys[key] = struct{}{}
			index = len(asksOfEachWordPartition)
			indexOfEachAddedWordPartition[key] = index
			asksOfEachWordPartition = append(asksOfEachWordPartition, nil)
		}
		asksOfEachWordPartition[index] = append(
			asksOfEachWordPartition[index],
			placedAsk[Ask]{
				ask:                 ask,
				replica:             replica,
				placeInReplicaOrder: len(asksOfEachWordPartition[index]),
			},
		)
	}

	return asksOfEachWordPartition
}

func wordPartitionKeyOf(replica Replica) wordPartitionKey {
	return wordPartitionKey{word: replica.Word.String(), partition: replica.Partition}
}

func (run *openRun[Ask, Answered]) startAsking(
	ctx context.Context,
	partition *wordPartition[Ask, Answered],
) {
	run.amountOfWordPartitionsUnsettled++
	go func() {
		partition.askUntilSettled(ctx)
		run.askedWordPartitions <- partition
	}()
}

func (run *openRun[Ask, Answered]) takeTheSettledWordPartition(
	partition *wordPartition[Ask, Answered],
) {
	run.amountOfWordPartitionsUnsettled--
	run.settledWordPartitionsUnread = append(
		run.settledWordPartitionsUnread,
		partition.settledWordPartition(),
	)
}

func (
	run *openRun[Ask, Answered],
) readerOfTheNextSettledWordPartition() chan<- SettledWordPartition[Ask, Answered] {
	if len(run.settledWordPartitionsUnread) == 0 {
		return nil
	}

	return run.settledWordPartitions
}

func (run *openRun[Ask, Answered]) nextSettledWordPartition() SettledWordPartition[Ask, Answered] {
	if len(run.settledWordPartitionsUnread) == 0 {
		return SettledWordPartition[Ask, Answered]{}
	}

	return run.settledWordPartitionsUnread[0]
}

func (run *openRun[Ask, Answered]) performedReplicaAsks() PerformedReplicaAsks {
	performedWordPartitions := make([]PerformedWordPartition, 0, len(run.wordPartitions))
	for _, partition := range run.wordPartitions {
		performedWordPartitions = append(
			performedWordPartitions,
			partition.performedWordPartition(),
		)
	}

	return PerformedReplicaAsks{
		EndedBy:        run.endedBy(),
		TimeSpent:      time.Since(run.startedAt),
		WordPartitions: performedWordPartitions,
	}
}

func (run *openRun[Ask, Answered]) endedBy() EndedBy {
	for _, partition := range run.wordPartitions {
		if partition.settledBy == SettledByDeadline {
			return EndedByDeadline
		}
	}

	return EndedByCoverage
}
