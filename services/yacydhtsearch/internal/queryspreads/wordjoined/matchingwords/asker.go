// Package matchingwords asks for the matching words of a query, partition by
// partition, naming the documents they must match where it can.
package matchingwords

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/wordroles"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type WordAsks interface {
	PartitionSettledFor(partition uint, words []yacymodel.Hash) bool
	PartitionAskedFor(partition uint, words []yacymodel.Hash) bool
	WaitUntilAnyPartitionSettles()
	DocumentsListedIn(partition uint, words []yacymodel.Hash) yacymodel.URLHashes
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

func (asker Asker) AskInEachPartition(
	roles wordroles.Roles,
	amountOfDocumentsToMatchInAPartition yacymodel.Optional[int],
	wordAsks WordAsks,
) KindPerPartition {
	if asker.predictsOverTheCeiling(amountOfDocumentsToMatchInAPartition, roles.MatchingWords) {
		return asker.askEveryPartitionNow(roles, wordAsks)
	}

	return asker.askEachPartitionAsItSettles(roles, wordAsks)
}

func (asker Asker) predictsOverTheCeiling(
	amountOfDocumentsToMatchInAPartition yacymodel.Optional[int],
	matchingWords []yacymodel.Hash,
) bool {
	amountOfDocuments, counted := amountOfDocumentsToMatchInAPartition.Get()

	return counted && len(matchingWords) > 0 && amountOfDocuments > asker.documentsToMatchCeiling
}

func (asker Asker) askEveryPartitionNow(roles wordroles.Roles, wordAsks WordAsks) KindPerPartition {
	wordAsks.AskEveryPartitionFor(roles.MatchingWords)
	kindPerPartition := KindPerPartition{}
	for _, partition := range asker.partitionsOfTheRing() {
		kindPerPartition[partition] = PredictedOverTheCeiling
	}

	return kindPerPartition
}

func (asker Asker) partitionsOfTheRing() []uint {
	partitionsOfTheRing := make([]uint, 0, asker.partitions)
	for partition := range uint(asker.partitions) {
		partitionsOfTheRing = append(partitionsOfTheRing, partition)
	}

	return partitionsOfTheRing
}

func (asker Asker) askEachPartitionAsItSettles(
	roles wordroles.Roles,
	wordAsks WordAsks,
) KindPerPartition {
	kindPerPartition := KindPerPartition{}
	partitionsLeft := partitionsLeft(asker.partitionsOfTheRing())
	for {
		settledPartitions := partitionsLeft.settledFor(roles.ListingWords, wordAsks)
		for _, partition := range settledPartitions {
			if kind, asked := asker.askIn(partition, roles, wordAsks).Get(); asked {
				kindPerPartition[partition] = kind
			}
		}
		partitionsLeft = partitionsLeft.without(settledPartitions)
		if len(partitionsLeft) == 0 {
			return kindPerPartition
		}
		wordAsks.WaitUntilAnyPartitionSettles()
	}
}

type partitionsLeft []uint

func (partitions partitionsLeft) settledFor(
	listingWords []yacymodel.Hash,
	wordAsks WordAsks,
) []uint {
	var settledPartitions []uint
	for _, partition := range partitions {
		if wordAsks.PartitionSettledFor(partition, listingWords) {
			settledPartitions = append(settledPartitions, partition)
		}
	}

	return settledPartitions
}

func (partitions partitionsLeft) without(settledPartitions []uint) partitionsLeft {
	return slices.DeleteFunc(partitions, func(partition uint) bool {
		return slices.Contains(settledPartitions, partition)
	})
}

func (asker Asker) askIn(
	partition uint,
	roles wordroles.Roles,
	wordAsks WordAsks,
) yacymodel.Optional[Kind] {
	if wordAsks.PartitionAskedFor(partition, roles.MatchingWords) {
		return yacymodel.None[Kind]()
	}
	documentsListed := wordAsks.DocumentsListedIn(partition, roles.ListingWords)
	documentsToMatch := documentsListed.InHashOrder()
	switch {
	case len(documentsToMatch) == 0:
		return yacymodel.Some(Skipped)
	case len(documentsToMatch) > asker.documentsToMatchCeiling:
		wordAsks.AskPartitionFor(partition, roles.MatchingWords)

		return yacymodel.Some(OverTheCeiling)
	default:
		wordAsks.AskPartitionForWordsAmong(partition, roles.MatchingWords, documentsToMatch)

		return yacymodel.Some(NamingTheDocumentsToMatch)
	}
}
