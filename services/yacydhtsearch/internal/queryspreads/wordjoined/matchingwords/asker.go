// Package matchingwords asks for the matching words of a query, partition by
// partition, naming the documents they must match where it can.
package matchingwords

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/leadingword"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordroles"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Run interface {
	PartitionSettledFor(partition uint, words []yacymodel.Hash) bool
	PartitionAskedFor(partition uint, words []yacymodel.Hash) bool
	WaitUntilAnyPartitionSettles()
	DocumentsListedIn(partition uint, words []yacymodel.Hash) yacymodel.URLHashes
	DocumentHolders() documentholders.Holders
	AskEveryPartitionFor(words []yacymodel.Hash)
	AskPartitionFor(partition uint, words []yacymodel.Hash)
	AskPartitionForWordsAmong(
		partition uint,
		words []yacymodel.Hash,
		documentsToMatch []yacymodel.URLHash,
	)
}

type Asker struct {
	partitions              yacymodel.DHTRingPartitions
	documentsToMatchCeiling int
}

func New(partitions yacymodel.DHTRingPartitions, documentsToMatchCeiling int) Asker {
	return Asker{partitions: partitions, documentsToMatchCeiling: documentsToMatchCeiling}
}

func (asker Asker) AskFor(roles wordroles.Roles, lead leadingword.Lead, run Run) AsksPerPartition {
	if asker.predictsOverTheCeiling(lead, roles.MatchingWords) {
		return asker.askEveryPartitionFor(roles.MatchingWords, run)
	}

	return asker.askAsEachPartitionSettles(roles, run)
}

func (asker Asker) predictsOverTheCeiling(
	lead leadingword.Lead,
	matchingWords []yacymodel.Hash,
) bool {
	amountOfDocuments, counted := lead.AmountOfDocumentsInAPartition.Get()

	return counted && len(matchingWords) > 0 && amountOfDocuments > asker.documentsToMatchCeiling
}

func (asker Asker) askEveryPartitionFor(
	matchingWords []yacymodel.Hash,
	run Run,
) AsksPerPartition {
	run.AskEveryPartitionFor(matchingWords)
	asksPerPartition := AsksPerPartition{}
	for _, partition := range asker.partitionsOfTheRing() {
		asksPerPartition[partition] = PredictedOverTheCeiling
	}

	return asksPerPartition
}

func (asker Asker) partitionsOfTheRing() []uint {
	partitionsOfTheRing := make([]uint, 0, asker.partitions)
	for partition := range uint(asker.partitions) {
		partitionsOfTheRing = append(partitionsOfTheRing, partition)
	}

	return partitionsOfTheRing
}

func (asker Asker) askAsEachPartitionSettles(roles wordroles.Roles, run Run) AsksPerPartition {
	asksPerPartition := AsksPerPartition{}
	partitionsLeft := asker.partitionsOfTheRing()
	for {
		settledPartitions := partitionsSettledAmong(partitionsLeft, roles.ListingWords, run)
		for _, partition := range settledPartitions {
			if kind, asked := asker.askIn(partition, roles, run).Get(); asked {
				asksPerPartition[partition] = kind
			}
		}
		partitionsLeft = partitionsWithout(partitionsLeft, settledPartitions)
		if len(partitionsLeft) == 0 {
			return asksPerPartition
		}
		run.WaitUntilAnyPartitionSettles()
	}
}

func partitionsSettledAmong(partitions []uint, listingWords []yacymodel.Hash, run Run) []uint {
	var settledPartitions []uint
	for _, partition := range partitions {
		if run.PartitionSettledFor(partition, listingWords) {
			settledPartitions = append(settledPartitions, partition)
		}
	}

	return settledPartitions
}

func (asker Asker) askIn(
	partition uint,
	roles wordroles.Roles,
	run Run,
) yacymodel.Optional[Kind] {
	if run.PartitionAskedFor(partition, roles.MatchingWords) {
		return yacymodel.None[Kind]()
	}
	holders := run.DocumentHolders()
	documentsToMatchMostHeldFirst := holders.MostHeldFirst(
		run.DocumentsListedIn(partition, roles.ListingWords),
	)
	kind := kindFrom(documentsToMatchMostHeldFirst, asker.documentsToMatchCeiling)
	kind.askIn(partition, roles.MatchingWords, documentsToMatchMostHeldFirst, run)

	return yacymodel.Some(kind)
}

func partitionsWithout(partitions []uint, partitionsLeftOut []uint) []uint {
	return slices.DeleteFunc(partitions, func(partition uint) bool {
		return slices.Contains(partitionsLeftOut, partition)
	})
}
