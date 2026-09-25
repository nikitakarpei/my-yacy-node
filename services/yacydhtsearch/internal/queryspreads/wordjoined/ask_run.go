package wordjoined

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type askRun struct {
	asks                  chan<- []peerasks.WordAbstractAsk
	settledWordPartitions <-chan replicaasks.SettledWordPartition[
		peerasks.WordAbstractAsk, peerasks.AnsweredWordAbstractAsk,
	]
	askOutcomes              peerasks.WordAbstractAskOutcomes
	holdersPerDocument       holdersPerDocument
	askedWordPartitionKeys   map[wordPartitionKey]struct{}
	settledWordPartitionKeys map[wordPartitionKey]struct{}
}

type wordPartitionKey struct {
	word      yacymodel.Hash
	partition uint
}

func startAskRun(ctx context.Context, replicaAsks ReplicaAsks) *askRun {
	run := replicaAsks.Start(ctx)

	return &askRun{
		asks:                     run.Asks,
		settledWordPartitions:    run.SettledWordPartitions,
		holdersPerDocument:       holdersPerDocument{},
		askedWordPartitionKeys:   map[wordPartitionKey]struct{}{},
		settledWordPartitionKeys: map[wordPartitionKey]struct{}{},
	}
}

func (askRun *askRun) put(asks discoveryAsks) {
	if len(asks) == 0 {
		return
	}
	for _, ask := range asks {
		askRun.askedWordPartitionKeys[wordPartitionKeyOf(ask)] = struct{}{}
	}
	askRun.asks <- asks
}

func wordPartitionKeyOf(ask peerasks.WordAbstractAsk) wordPartitionKey {
	return wordPartitionKey{word: ask.Word, partition: ask.Partition}
}

func (askRun *askRun) notAskedAmong(asks discoveryAsks) discoveryAsks {
	var asksNotAsked discoveryAsks
	for _, ask := range asks {
		if _, asked := askRun.askedWordPartitionKeys[wordPartitionKeyOf(ask)]; asked {
			continue
		}
		asksNotAsked = append(asksNotAsked, ask)
	}

	return asksNotAsked
}

func (askRun *askRun) readUntilSettled(asks discoveryAsks) {
	for !askRun.haveSettled(asks) {
		askRun.readTheNextSettledWordPartition()
	}
}

func (askRun *askRun) haveSettled(asks discoveryAsks) bool {
	for _, ask := range asks {
		if _, settled := askRun.settledWordPartitionKeys[wordPartitionKeyOf(ask)]; !settled {
			return false
		}
	}

	return true
}

func (askRun *askRun) readTheNextSettledWordPartition() {
	askRun.record(<-askRun.settledWordPartitions)
}

func (askRun *askRun) record(
	settledWordPartition replicaasks.SettledWordPartition[
		peerasks.WordAbstractAsk, peerasks.AnsweredWordAbstractAsk,
	],
) {
	askRun.askOutcomes = append(askRun.askOutcomes, settledWordPartition.AskOutcomes...)
	askRun.holdersPerDocument.addHoldersIn(settledWordPartition.AskOutcomes)
	for _, askOutcome := range settledWordPartition.AskOutcomes {
		askRun.settledWordPartitionKeys[wordPartitionKeyOf(askOutcome.Ask)] = struct{}{}
	}
}

func (askRun *askRun) finish() {
	close(askRun.asks)
	for settledWordPartition := range askRun.settledWordPartitions {
		askRun.record(settledWordPartition)
	}
}
