package documentasks

import (
	"context"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Inquiry struct {
	ctx                     context.Context
	chosenPeers             chosenPeers
	partitions              yacymodel.DHTRingPartitions
	documentsToMatchCeiling int
	observer                DocumentAsksObserver
	inquirer                Inquirer
	run                     wordpartitionasks.Run
	sentAsks                *sentAsks
	expectedInAPartition    map[yacymodel.Hash]int
}

func (inquiry *Inquiry) WhichDocumentsHave(words []yacymodel.Hash) {
	inquiry.askEveryPartitionFor(words)
	inquiry.waitUntilNonePendingFor(words)
}

func (inquiry *Inquiry) askEveryPartitionFor(words []yacymodel.Hash) {
	inquiry.send(inquiry.chosenPeers.asksOfEveryPartitionFor(words, inquiry.partitions))
}

func (inquiry *Inquiry) send(asks []wordpartitionasks.Ask) {
	asksToSend := slices.DeleteFunc(asks, inquiry.sentAsks.contain)
	if len(asksToSend) == 0 {
		return
	}
	inquiry.sentAsks.add(asksToSend)
	inquiry.run.Asks <- asksToSend
}

func (inquiry *Inquiry) waitUntilNonePendingFor(words []yacymodel.Hash) {
	for !inquiry.sentAsks.nonePendingFor(words) {
		inquiry.readTheNextSettledAsk()
	}
}

func (inquiry *Inquiry) readTheNextSettledAsk() {
	inquiry.settle(<-inquiry.run.SettledAsks)
}

func (inquiry *Inquiry) settle(settledAsk wordpartitionasks.SettledAsk) {
	inquiry.sentAsks.settle(settledAsk)
	inquiry.inquirer.WordPartitionAnswered(
		settledAsk.Word, settledAsk.Partition, settledAsk.Answers,
	)
}

type AnsweredWordPartition struct {
	Word      yacymodel.Hash
	Partition uint
	Answers   []wordpartitionasks.ReplicaAnswer
}

func (inquiry *Inquiry) WhichDocumentsHaveIn(
	partition uint,
	words []yacymodel.Hash,
) []AnsweredWordPartition {
	inquiry.send(inquiry.chosenPeers.asksOf(words, partition))
	inquiry.waitUntilNonePendingIn(partition, words)

	return inquiry.sentAsks.answersOf(partition, words)
}

func (inquiry *Inquiry) waitUntilNonePendingIn(partition uint, words []yacymodel.Hash) {
	for !inquiry.sentAsks.nonePendingIn(partition, words) {
		inquiry.readTheNextSettledAsk()
	}
}

func (inquiry *Inquiry) WhichDocumentsHavingTheseAlsoHave(these, those []yacymodel.Hash) {
	inquiry.askEveryPartitionFor(these)
	if len(those) > 0 {
		inquiry.observer.AskedAmongTheDocuments(
			inquiry.ctx, inquiry.askAmongTheDocumentsHaving(these, those),
		)
	}
	inquiry.waitUntilNonePendingFor(slices.Concat(these, those))
}

func (inquiry *Inquiry) askAmongTheDocumentsHaving(
	these, those []yacymodel.Hash,
) DocumentsToMatchDecisionPerPartition {
	if inquiry.expectedOverTheCeiling(these) {
		inquiry.askEveryPartitionFor(those)

		return inquiry.everyPartition(NamedNonePredictedOverTheCeiling)
	}
	decisionPerPartition := DocumentsToMatchDecisionPerPartition{}
	inquiry.asEachPartitionListsDocumentsHaving(
		these,
		func(partition uint, listedDocuments yacymodel.URLHashes) {
			decided := inquiry.askWhetherTheyAlsoHave(partition, listedDocuments, those)
			if decision, asked := decided.Get(); asked {
				decisionPerPartition[partition] = decision
			}
		},
	)

	return decisionPerPartition
}

func (inquiry *Inquiry) expectedOverTheCeiling(words []yacymodel.Hash) bool {
	for _, word := range words {
		if inquiry.expectedInAPartition[word] > inquiry.documentsToMatchCeiling {
			return true
		}
	}

	return false
}

func (inquiry *Inquiry) everyPartition(
	decision DocumentsToMatchDecision,
) DocumentsToMatchDecisionPerPartition {
	decisionPerPartition := DocumentsToMatchDecisionPerPartition{}
	for partition := range uint(inquiry.partitions) {
		decisionPerPartition[partition] = decision
	}

	return decisionPerPartition
}

func (inquiry *Inquiry) asEachPartitionListsDocumentsHaving(
	words []yacymodel.Hash,
	then func(partition uint, listedDocuments yacymodel.URLHashes),
) {
	partitionsLeft := inquiry.partitionsOfTheRing()
	for len(partitionsLeft) > 0 {
		listingPartitions := inquiry.partitionsThatListedDocumentsHaving(words, partitionsLeft)
		for _, partition := range listingPartitions {
			then(partition, inquiry.sentAsks.documentsListedIn(partition, words))
		}
		partitionsLeft = partitionsWithout(partitionsLeft, listingPartitions)
		if len(partitionsLeft) > 0 {
			inquiry.readTheNextSettledAsk()
		}
	}
}

func (inquiry *Inquiry) partitionsOfTheRing() []uint {
	partitionsOfTheRing := make([]uint, 0, inquiry.partitions)
	for partition := range uint(inquiry.partitions) {
		partitionsOfTheRing = append(partitionsOfTheRing, partition)
	}

	return partitionsOfTheRing
}

func (inquiry *Inquiry) partitionsThatListedDocumentsHaving(
	words []yacymodel.Hash,
	partitions []uint,
) []uint {
	var listingPartitions []uint
	for _, partition := range partitions {
		if inquiry.sentAsks.nonePendingIn(partition, words) {
			listingPartitions = append(listingPartitions, partition)
		}
	}

	return listingPartitions
}

func partitionsWithout(partitions []uint, excludedPartitions []uint) []uint {
	return slices.DeleteFunc(partitions, func(partition uint) bool {
		return slices.Contains(excludedPartitions, partition)
	})
}

func (inquiry *Inquiry) askWhetherTheyAlsoHave(
	partition uint,
	listedDocuments yacymodel.URLHashes,
	words []yacymodel.Hash,
) yacymodel.Optional[DocumentsToMatchDecision] {
	asks := inquiry.chosenPeers.asksOf(words, partition)
	if inquiry.sentAsks.containEvery(asks) {
		return yacymodel.None[DocumentsToMatchDecision]()
	}
	documentsToMatch := listedDocuments.InHashOrder()
	switch {
	case len(documentsToMatch) == 0:
		return yacymodel.Some(NoDocumentsToMatch)
	case len(documentsToMatch) > inquiry.documentsToMatchCeiling:
		inquiry.send(asks)

		return yacymodel.Some(NamedNoneOverTheCeiling)
	default:
		inquiry.send(namingTheDocumentsToMatch(asks, documentsToMatch))

		return yacymodel.Some(NamedTheDocumentsToMatch)
	}
}

func (inquiry *Inquiry) ExpectInAPartition(word yacymodel.Hash, amountOfDocuments int) {
	inquiry.expectedInAPartition[word] = amountOfDocuments
}

func (inquiry *Inquiry) End() []AnsweredWordPartition {
	close(inquiry.run.Asks)
	for settledAsk := range inquiry.run.SettledAsks {
		inquiry.settle(settledAsk)
	}
	inquiry.observer.DocumentAsksPerformed(inquiry.ctx, performedFrom(inquiry.sentAsks.answered))

	return inquiry.sentAsks.answered
}
