// Package documentasks asks the partitions which documents have which words, for
// one query, from the first question of its inquiry until End, which returns all
// it was answered. A question returns once its words are answered, and the answers
// of each ask go to the inquirer as the ask settles. Which documents have words in
// one partition answers with what its replicas listed, whenever they were asked.
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

type Inquirer interface {
	WordPartitionAnswered(
		word yacymodel.Hash,
		partition uint,
		answers []wordpartitionasks.ReplicaAnswer,
	)
}

type Inquirers []Inquirer

func (inquirers Inquirers) WordPartitionAnswered(
	word yacymodel.Hash,
	partition uint,
	answers []wordpartitionasks.ReplicaAnswer,
) {
	for _, inquirer := range inquirers {
		inquirer.WordPartitionAnswered(word, partition, answers)
	}
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
	inquirer Inquirer,
) *Inquiry {
	return &Inquiry{
		ctx:                     ctx,
		chosenPeers:             chosenPeersFor(query, chosenPeersPerQueryWord),
		partitions:              asker.partitions,
		documentsToMatchCeiling: asker.documentsToMatchCeiling,
		observer:                asker.observer,
		inquirer:                inquirer,
		run:                     asker.wordPartitionAsks.Start(ctx),
		sentAsks:                noAsksSentYet(asker.partitions),
		expectedInAPartition:    map[yacymodel.Hash]int{},
	}
}
