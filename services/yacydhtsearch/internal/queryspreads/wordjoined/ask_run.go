package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type askRun struct {
	run                   replicaasks.Run
	askOutcomes           peerasks.SearchDocumentsAskOutcomes
	putWordPartitions     map[wordPartitionKey]struct{}
	settledWordPartitions map[wordPartitionKey]struct{}
}

type wordPartitionKey struct {
	word      yacymodel.Hash
	partition uint
}

func askRunOf(run replicaasks.Run) *askRun {
	return &askRun{
		run:                   run,
		putWordPartitions:     map[wordPartitionKey]struct{}{},
		settledWordPartitions: map[wordPartitionKey]struct{}{},
	}
}

func (askRun *askRun) put(asks discoveryAsks) {
	if len(asks) == 0 {
		return
	}
	for _, ask := range asks {
		askRun.putWordPartitions[wordPartitionKeyOf(ask)] = struct{}{}
	}
	askRun.run.Asks <- asks
}

func wordPartitionKeyOf(ask peerasks.SearchDocumentsAsk) wordPartitionKey {
	return wordPartitionKey{word: ask.Word, partition: ask.Partition}
}

func (askRun *askRun) notPutAmong(asks discoveryAsks) discoveryAsks {
	var asksNotPut discoveryAsks
	for _, ask := range asks {
		if _, put := askRun.putWordPartitions[wordPartitionKeyOf(ask)]; put {
			continue
		}
		asksNotPut = append(asksNotPut, ask)
	}

	return asksNotPut
}

func (askRun *askRun) readUntilSettled(asks discoveryAsks) {
	for !askRun.haveSettled(asks) {
		askRun.readTheNextSettledWordPartition()
	}
}

func (askRun *askRun) haveSettled(asks discoveryAsks) bool {
	for _, ask := range asks {
		if _, settled := askRun.settledWordPartitions[wordPartitionKeyOf(ask)]; !settled {
			return false
		}
	}

	return true
}

func (askRun *askRun) readTheNextSettledWordPartition() {
	askRun.record(<-askRun.run.SettledWordPartitions)
}

func (askRun *askRun) record(settledWordPartition replicaasks.SettledWordPartition) {
	askRun.askOutcomes = append(askRun.askOutcomes, settledWordPartition.AskOutcomes...)
	for _, askOutcome := range settledWordPartition.AskOutcomes {
		askRun.settledWordPartitions[wordPartitionKeyOf(askOutcome.Ask)] = struct{}{}
	}
}

func (askRun *askRun) finish() {
	close(askRun.run.Asks)
	for settledWordPartition := range askRun.run.SettledWordPartitions {
		askRun.record(settledWordPartition)
	}
}
