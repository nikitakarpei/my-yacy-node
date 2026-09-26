package wordpartitionasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type openRun struct {
	replicaAsks                     Asks
	startedAt                       time.Time
	asks                            <-chan []peerasks.SearchDocumentsAsk
	settledWordPartitions           chan<- SettledWordPartition
	askedWordPartitions             chan *wordPartition
	chosenPeers                     *chosenPeers
	wordPartitions                  []*wordPartition
	wordPartitionKeys               map[wordPartitionKey]struct{}
	amountOfWordPartitionsUnsettled int
	settledWordPartitionsUnread     []SettledWordPartition
}

type wordPartitionKey struct {
	word      string
	partition uint
}

func openRunOf(
	replicaAsks Asks,
	asks <-chan []peerasks.SearchDocumentsAsk,
	settledWordPartitions chan<- SettledWordPartition,
) *openRun {
	return &openRun{
		replicaAsks:           replicaAsks,
		startedAt:             time.Now(),
		asks:                  asks,
		settledWordPartitions: settledWordPartitions,
		askedWordPartitions:   make(chan *wordPartition),
		chosenPeers:           noChosenPeers(),
		wordPartitionKeys:     map[wordPartitionKey]struct{}{},
	}
}

func (run *openRun) askUntilOver(ctx context.Context) {
	for !run.isOver() {
		run.takeTheNextEvent(ctx)
	}
	run.replicaAsks.observer.ReplicaAsksPerformed(ctx, run.performedReplicaAsks())
	close(run.settledWordPartitions)
}

func (run *openRun) isOver() bool {
	return run.asks == nil && run.amountOfWordPartitionsUnsettled == 0 &&
		len(run.settledWordPartitionsUnread) == 0
}

func (run *openRun) takeTheNextEvent(ctx context.Context) {
	select {
	case asks, open := <-run.asks:
		run.takeTheAsks(ctx, asks, open)
	case partition := <-run.askedWordPartitions:
		run.takeTheSettledWordPartition(partition)
	case run.readerOfTheNextSettledWordPartition() <- run.nextSettledWordPartition():
		run.settledWordPartitionsUnread = run.settledWordPartitionsUnread[1:]
	}
}

func (run *openRun) takeTheAsks(
	ctx context.Context,
	asksInReplicaOrder []peerasks.SearchDocumentsAsk,
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

func (run *openRun) wordPartitionsOf(
	asksInReplicaOrder []peerasks.SearchDocumentsAsk,
) []*wordPartition {
	asksOfEachWordPartition := run.asksOfEachAddedWordPartitionIn(asksInReplicaOrder)
	addedWordPartitions := make([]*wordPartition, 0, len(asksOfEachWordPartition))
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

func (run *openRun) asksOfEachAddedWordPartitionIn(
	asksInReplicaOrder []peerasks.SearchDocumentsAsk,
) [][]placedAsk {
	asksOfEachWordPartition := make([][]placedAsk, 0, len(asksInReplicaOrder))
	indexOfEachAddedWordPartition := make(map[wordPartitionKey]int, len(asksInReplicaOrder))
	for _, ask := range asksInReplicaOrder {
		key := wordPartitionKeyOf(ask)
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
			placedAsk{ask: ask, placeInReplicaOrder: len(asksOfEachWordPartition[index])},
		)
	}

	return asksOfEachWordPartition
}

func wordPartitionKeyOf(ask peerasks.SearchDocumentsAsk) wordPartitionKey {
	return wordPartitionKey{word: ask.Word.String(), partition: ask.Partition}
}

func (run *openRun) startAsking(ctx context.Context, partition *wordPartition) {
	run.amountOfWordPartitionsUnsettled++
	go func() {
		partition.askUntilSettled(ctx)
		run.askedWordPartitions <- partition
	}()
}

func (run *openRun) takeTheSettledWordPartition(partition *wordPartition) {
	run.amountOfWordPartitionsUnsettled--
	run.settledWordPartitionsUnread = append(
		run.settledWordPartitionsUnread,
		partition.settledWordPartition(),
	)
}

func (run *openRun) readerOfTheNextSettledWordPartition() chan<- SettledWordPartition {
	if len(run.settledWordPartitionsUnread) == 0 {
		return nil
	}

	return run.settledWordPartitions
}

func (run *openRun) nextSettledWordPartition() SettledWordPartition {
	if len(run.settledWordPartitionsUnread) == 0 {
		return SettledWordPartition{}
	}

	return run.settledWordPartitionsUnread[0]
}

func (run *openRun) performedReplicaAsks() PerformedReplicaAsks {
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

func (run *openRun) endedBy() EndedBy {
	for _, partition := range run.wordPartitions {
		if partition.settledBy == SettledByDeadline {
			return EndedByDeadline
		}
	}

	return EndedByCoverage
}
