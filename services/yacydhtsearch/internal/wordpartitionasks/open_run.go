package wordpartitionasks

import (
	"context"
	"time"
)

type openRun struct {
	replicaAsks                     Asks
	startedAt                       time.Time
	asks                            <-chan []Ask
	settledAsks                     chan<- SettledAsk
	askedWordPartitions             chan *wordPartition
	chosenPeers                     *chosenPeers
	wordPartitions                  []*wordPartition
	wordPartitionKeys               map[wordPartitionKey]struct{}
	amountOfWordPartitionsUnsettled int
	settledAsksUnread               []SettledAsk
}

type wordPartitionKey struct {
	word      string
	partition uint
}

func openRunOf(
	replicaAsks Asks,
	asks <-chan []Ask,
	settledAsks chan<- SettledAsk,
) *openRun {
	return &openRun{
		replicaAsks:         replicaAsks,
		startedAt:           time.Now(),
		asks:                asks,
		settledAsks:         settledAsks,
		askedWordPartitions: make(chan *wordPartition),
		chosenPeers:         noChosenPeers(),
		wordPartitionKeys:   map[wordPartitionKey]struct{}{},
	}
}

func (run *openRun) askUntilOver(ctx context.Context) {
	for !run.isOver() {
		run.takeTheNextEvent(ctx)
	}
	run.replicaAsks.reportPerformed(ctx, run.performedReplicaAsks())
	close(run.settledAsks)
}

func (run *openRun) isOver() bool {
	return run.asks == nil && run.amountOfWordPartitionsUnsettled == 0 &&
		len(run.settledAsksUnread) == 0
}

func (run *openRun) takeTheNextEvent(ctx context.Context) {
	select {
	case asks, open := <-run.asks:
		run.takeTheAsks(ctx, asks, open)
	case partition := <-run.askedWordPartitions:
		run.takeTheSettledWordPartition(partition)
	case run.readerOfTheNextSettledAsk() <- run.nextSettledAsk():
		run.settledAsksUnread = run.settledAsksUnread[1:]
	}
}

func (run *openRun) takeTheAsks(ctx context.Context, asks []Ask, open bool) {
	if !open {
		run.asks = nil

		return
	}
	addedWordPartitions := run.wordPartitionsOf(asks)
	for _, partition := range addedWordPartitions {
		partition.chooseTheFirstReplicas()
	}
	for _, partition := range addedWordPartitions {
		run.startAsking(ctx, partition)
	}
}

func (run *openRun) wordPartitionsOf(asks []Ask) []*wordPartition {
	addedWordPartitions := make([]*wordPartition, 0, len(asks))
	for _, ask := range asks {
		key := wordPartitionKeyOf(ask)
		if _, alreadyInTheRun := run.wordPartitionKeys[key]; alreadyInTheRun {
			continue
		}
		run.wordPartitionKeys[key] = struct{}{}
		addedWordPartitions = append(
			addedWordPartitions,
			wordPartitionOf(ask, run.replicaAsks, run.chosenPeers),
		)
	}
	run.wordPartitions = append(run.wordPartitions, addedWordPartitions...)

	return addedWordPartitions
}

func wordPartitionKeyOf(ask Ask) wordPartitionKey {
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
	run.settledAsksUnread = append(run.settledAsksUnread, partition.settledAsk())
}

func (run *openRun) readerOfTheNextSettledAsk() chan<- SettledAsk {
	if len(run.settledAsksUnread) == 0 {
		return nil
	}

	return run.settledAsks
}

func (run *openRun) nextSettledAsk() SettledAsk {
	if len(run.settledAsksUnread) == 0 {
		return SettledAsk{}
	}

	return run.settledAsksUnread[0]
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
