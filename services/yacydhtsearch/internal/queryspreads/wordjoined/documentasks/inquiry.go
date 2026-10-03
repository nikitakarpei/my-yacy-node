package documentasks

import (
	"context"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Inquiry struct {
	ctx                     context.Context
	chosenReplicas          chosenReplicas
	partitions              yacymodel.DHTRingPartitions
	documentsToMatchCeiling int
	observer                DocumentAsksObserver
	asksToPut               chan<- []wordpartitionasks.Ask
	settledAsksAsTheySettle <-chan wordpartitionasks.SettledAsk
	stateOfEachSentAsk      map[wordPartitionKey]askState
	settledAsks             []wordpartitionasks.SettledAsk
}

type askState int

const (
	open askState = iota
	settled
)

func (inquiry *Inquiry) WhichDocumentsHave(words []yacymodel.Hash) {
	inquiry.send(inquiry.chosenReplicas.asksOfEveryPartitionFor(words, inquiry.partitions))
}

func (inquiry *Inquiry) send(asks []wordpartitionasks.Ask) {
	asksNotSent := slices.DeleteFunc(asks, inquiry.sent)
	if len(asksNotSent) == 0 {
		return
	}
	for _, ask := range asksNotSent {
		inquiry.stateOfEachSentAsk[wordPartitionKeyOf(ask)] = open
	}
	inquiry.asksToPut <- asksNotSent
}

func (inquiry *Inquiry) sent(ask wordpartitionasks.Ask) bool {
	_, sent := inquiry.stateOfEachSentAsk[wordPartitionKeyOf(ask)]

	return sent
}

func wordPartitionKeyOf(ask wordpartitionasks.Ask) wordPartitionKey {
	return wordPartitionKey{word: ask.Word, partition: ask.Partition}
}

func (inquiry *Inquiry) WhichDocumentsHaveIn(partition uint, words []yacymodel.Hash) {
	inquiry.send(inquiry.chosenReplicas.asksOf(words, partition))
}

func (inquiry *Inquiry) WhichDocumentsHavingTheseAlsoHave(
	these []yacymodel.Hash,
	those []yacymodel.Hash,
	amountOfTheseInAPartition int,
) {
	if len(those) > 0 && amountOfTheseInAPartition > inquiry.documentsToMatchCeiling {
		inquiry.askEveryPartitionNow(those)

		return
	}
	inquiry.askEachPartitionAsItSettles(these, those)
}

func (inquiry *Inquiry) askEveryPartitionNow(those []yacymodel.Hash) {
	inquiry.WhichDocumentsHave(those)
	for _, partition := range inquiry.partitionsOfTheRing() {
		inquiry.observer.AskedPartitionFor(inquiry.ctx, partition, PredictedOverTheCeiling)
	}
}

func (inquiry *Inquiry) partitionsOfTheRing() []uint {
	partitionsOfTheRing := make([]uint, 0, inquiry.partitions)
	for partition := range uint(inquiry.partitions) {
		partitionsOfTheRing = append(partitionsOfTheRing, partition)
	}

	return partitionsOfTheRing
}

func (inquiry *Inquiry) askEachPartitionAsItSettles(these, those []yacymodel.Hash) {
	partitionsLeftToAsk := partitionsLeft(inquiry.partitionsOfTheRing())
	for {
		settledPartitions := inquiry.settledFor(these, partitionsLeftToAsk)
		for _, partition := range settledPartitions {
			if kind, asked := inquiry.askIn(partition, these, those).Get(); asked {
				inquiry.observer.AskedPartitionFor(inquiry.ctx, partition, kind)
			}
		}
		partitionsLeftToAsk = partitionsLeftToAsk.without(settledPartitions)
		if len(partitionsLeftToAsk) == 0 {
			return
		}
		inquiry.readTheNextSettledAsk()
	}
}

func (inquiry *Inquiry) settledFor(words []yacymodel.Hash, partitions partitionsLeft) []uint {
	var settledPartitions []uint
	for _, partition := range partitions {
		if inquiry.partitionSettledFor(partition, words) {
			settledPartitions = append(settledPartitions, partition)
		}
	}

	return settledPartitions
}

func (inquiry *Inquiry) partitionSettledFor(partition uint, words []yacymodel.Hash) bool {
	for _, word := range words {
		state, sent := inquiry.stateOfEachSentAsk[wordPartitionKey{word: word, partition: partition}]
		if sent && state == open {
			return false
		}
	}

	return true
}

func (inquiry *Inquiry) askIn(
	partition uint,
	these []yacymodel.Hash,
	those []yacymodel.Hash,
) yacymodel.Optional[Kind] {
	asksOfThose := inquiry.chosenReplicas.asksOf(those, partition)
	if !slices.ContainsFunc(asksOfThose, func(ask wordpartitionasks.Ask) bool {
		return !inquiry.sent(ask)
	}) {
		return yacymodel.None[Kind]()
	}
	documentsToMatch := inquiry.documentsListedIn(partition, these).InHashOrder()
	switch {
	case len(documentsToMatch) == 0:
		return yacymodel.Some(Skipped)
	case len(documentsToMatch) > inquiry.documentsToMatchCeiling:
		inquiry.send(asksOfThose)

		return yacymodel.Some(OverTheCeiling)
	default:
		inquiry.send(withDocumentsToMatch(asksOfThose, documentsToMatch))

		return yacymodel.Some(NamingTheDocumentsToMatch)
	}
}

func (inquiry *Inquiry) documentsListedIn(
	partition uint,
	words []yacymodel.Hash,
) yacymodel.URLHashes {
	documents := yacymodel.URLHashes{}
	for _, settledAsk := range inquiry.settledAsks {
		if !slices.Contains(words, settledAsk.Word) {
			continue
		}
		for _, answer := range settledAsk.Answers {
			for _, listedDocument := range answer.ListedDocuments {
				if inquiry.partitions.PartitionOf(listedDocument.Hash) != partition {
					continue
				}
				documents.Add(listedDocument.Hash)
			}
		}
	}

	return documents
}

func (inquiry *Inquiry) readTheNextSettledAsk() {
	inquiry.record(<-inquiry.settledAsksAsTheySettle)
}

func (inquiry *Inquiry) record(settledAsk wordpartitionasks.SettledAsk) {
	inquiry.settledAsks = append(inquiry.settledAsks, settledAsk)
	inquiry.stateOfEachSentAsk[wordPartitionKeyOf(settledAsk.Ask)] = settled
}

func (inquiry *Inquiry) WaitUntilPartitionSettledFor(partition uint, words []yacymodel.Hash) {
	for !inquiry.partitionSettledFor(partition, words) {
		inquiry.readTheNextSettledAsk()
	}
}

func (inquiry *Inquiry) SettledIn(partition uint, words []yacymodel.Hash) Answers {
	return Answers{
		SettledAsks: slices.DeleteFunc(
			slices.Clone(inquiry.settledAsks),
			func(settledAsk wordpartitionasks.SettledAsk) bool {
				return settledAsk.Partition != partition || !slices.Contains(words, settledAsk.Word)
			},
		),
		Partitions: inquiry.partitions,
	}
}

func (inquiry *Inquiry) Finish() Answers {
	close(inquiry.asksToPut)
	for settledAsk := range inquiry.settledAsksAsTheySettle {
		inquiry.record(settledAsk)
	}

	return Answers{SettledAsks: inquiry.settledAsks, Partitions: inquiry.partitions}
}
