package wordjoined

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type askRun struct {
	asks                     chan<- []wordpartitionasks.Ask
	settledAsksAsTheySettle  <-chan wordpartitionasks.SettledAsk
	settledAsks              settledAsks
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
		settledAsksAsTheySettle:  run.SettledAsks,
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

func wordPartitionKeyOf(ask wordpartitionasks.Ask) wordPartitionKey {
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
		askRun.readTheNextSettledAsk()
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

func (askRun *askRun) readTheNextSettledAsk() {
	askRun.record(<-askRun.settledAsksAsTheySettle)
}

func (askRun *askRun) record(settledAsk wordpartitionasks.SettledAsk) {
	askRun.settledAsks = append(askRun.settledAsks, settledAsk)
	askRun.holdersPerDocument.addHoldersIn(settledAsk.Answers)
	askRun.settledWordPartitionKeys[wordPartitionKeyOf(settledAsk.Ask)] = struct{}{}
}

func (askRun *askRun) finish() {
	close(askRun.asks)
	for settledAsk := range askRun.settledAsksAsTheySettle {
		askRun.record(settledAsk)
	}
}
