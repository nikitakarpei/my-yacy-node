// Package documentasks asks the partitions which documents have which words, for
// one query. The asking done for one query is its inquiry: from the first
// question until Finish, with the answers collected along the way. One question
// is one call on the inquiry and may expand into many asks over the wire.
package documentasks

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type WordPartitionAsks interface {
	Start(ctx context.Context) wordpartitionasks.Run
}

type Asker struct {
	wordPartitionAsks       WordPartitionAsks
	partitions              yacymodel.DHTRingPartitions
	documentsToMatchCeiling int
	observer                DocumentAsksObserver
}

func New(
	wordPartitionAsks WordPartitionAsks,
	partitions yacymodel.DHTRingPartitions,
	documentsToMatchCeiling int,
	observer DocumentAsksObserver,
) Asker {
	return Asker{
		wordPartitionAsks:       wordPartitionAsks,
		partitions:              partitions,
		documentsToMatchCeiling: documentsToMatchCeiling,
		observer:                observer,
	}
}

func (asker Asker) Begin(
	ctx context.Context,
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) *Inquiry {
	startedRun := asker.wordPartitionAsks.Start(ctx)

	return &Inquiry{
		ctx:                     ctx,
		chosenPeers:             chosenPeersFor(query, chosenPeersPerQueryWord),
		partitions:              asker.partitions,
		documentsToMatchCeiling: asker.documentsToMatchCeiling,
		observer:                asker.observer,
		asksToPut:               startedRun.Asks,
		settledAsksAsTheySettle: startedRun.SettledAsks,
		stateOfEachSentAsk:      map[wordPartitionKey]askState{},
	}
}
