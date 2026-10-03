// Package documentasks asks the partitions which documents have which words, for
// one query. The asking done for one query is its inquiry: from the first
// question until End. One question is one call on the inquiry, answered when it
// returns, and it may expand into many asks over the wire. Which
// documents having some words also have others is asked partition by partition,
// naming the documents the first words listed there, unless too many to name.
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
	return &Inquiry{
		ctx:                     ctx,
		chosenPeers:             chosenPeersFor(query, chosenPeersPerQueryWord),
		partitions:              asker.partitions,
		documentsToMatchCeiling: asker.documentsToMatchCeiling,
		observer:                asker.observer,
		run:                     asker.wordPartitionAsks.Start(ctx),
		sentAsks:                noAsksSentYet(asker.partitions),
		expectedInAPartition:    map[yacymodel.Hash]int{},
	}
}
