package wordasks

import (
	"context"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type WordPartitionAsks interface {
	Start(ctx context.Context) wordpartitionasks.Run
}

type Run struct {
	plannedAsks              asks
	partitions               yacymodel.DHTRingPartitions
	asksToPut                chan<- []wordpartitionasks.Ask
	settledAsksAsTheySettle  <-chan wordpartitionasks.SettledAsk
	answers                  Answers
	askedWordPartitionKeys   map[wordPartitionKey]struct{}
	settledWordPartitionKeys map[wordPartitionKey]struct{}
}

type wordPartitionKey struct {
	word      yacymodel.Hash
	partition uint
}

func Start(
	ctx context.Context,
	wordPartitionAsks WordPartitionAsks,
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
	partitions yacymodel.DHTRingPartitions,
) *Run {
	startedRun := wordPartitionAsks.Start(ctx)

	return &Run{
		plannedAsks:              asksFor(query, chosenPeersPerQueryWord),
		partitions:               partitions,
		asksToPut:                startedRun.Asks,
		settledAsksAsTheySettle:  startedRun.SettledAsks,
		askedWordPartitionKeys:   map[wordPartitionKey]struct{}{},
		settledWordPartitionKeys: map[wordPartitionKey]struct{}{},
	}
}

func (run *Run) AskEveryPartitionFor(words []yacymodel.Hash) {
	run.put(run.plannedAsks.ofWords(words))
}

func (run *Run) AskPartitionFor(partition uint, words []yacymodel.Hash) {
	run.put(run.plannedAsks.ofWordsIn(words, partition))
}

func (run *Run) AskPartitionForWordsAmong(
	partition uint,
	words []yacymodel.Hash,
	documentsToMatch []yacymodel.URLHash,
) {
	run.put(run.plannedAsks.ofWordsIn(words, partition).forDocumentsToMatch(documentsToMatch))
}

func (run *Run) put(asksOfTheWords asks) {
	asksNotAsked := slices.DeleteFunc(slices.Clone(asksOfTheWords), run.asked)
	if len(asksNotAsked) == 0 {
		return
	}
	for _, ask := range asksNotAsked {
		run.askedWordPartitionKeys[wordPartitionKeyOf(ask)] = struct{}{}
	}
	run.asksToPut <- asksNotAsked
}

func (run *Run) asked(ask wordpartitionasks.Ask) bool {
	_, asked := run.askedWordPartitionKeys[wordPartitionKeyOf(ask)]

	return asked
}

func wordPartitionKeyOf(ask wordpartitionasks.Ask) wordPartitionKey {
	return wordPartitionKey{word: ask.Word, partition: ask.Partition}
}

func (run *Run) PartitionAskedFor(partition uint, words []yacymodel.Hash) bool {
	return !slices.ContainsFunc(run.plannedAsks.ofWordsIn(words, partition), func(
		ask wordpartitionasks.Ask,
	) bool {
		return !run.asked(ask)
	})
}

func (run *Run) PartitionSettledFor(partition uint, words []yacymodel.Hash) bool {
	return !slices.ContainsFunc(run.plannedAsks.ofWordsIn(words, partition), func(
		ask wordpartitionasks.Ask,
	) bool {
		_, settled := run.settledWordPartitionKeys[wordPartitionKeyOf(ask)]

		return !settled
	})
}

func (run *Run) WaitUntilPartitionSettledFor(partition uint, words []yacymodel.Hash) {
	for !run.PartitionSettledFor(partition, words) {
		run.readTheNextSettledAsk()
	}
}

func (run *Run) WaitUntilAnyPartitionSettles() {
	run.readTheNextSettledAsk()
}

func (run *Run) readTheNextSettledAsk() {
	run.record(<-run.settledAsksAsTheySettle)
}

func (run *Run) record(settledAsk wordpartitionasks.SettledAsk) {
	run.answers = append(run.answers, settledAsk)
	run.settledWordPartitionKeys[wordPartitionKeyOf(settledAsk.Ask)] = struct{}{}
}

func (run *Run) SettledIn(
	partition uint,
	words []yacymodel.Hash,
) []wordpartitionasks.SettledAsk {
	return slices.DeleteFunc(
		slices.Clone(run.answers),
		func(settledAsk wordpartitionasks.SettledAsk) bool {
			return settledAsk.Partition != partition || !slices.Contains(words, settledAsk.Word)
		},
	)
}

func (run *Run) DocumentsListedIn(
	partition uint,
	words []yacymodel.Hash,
) yacymodel.URLHashes {
	documents := yacymodel.URLHashes{}
	for _, settledAsk := range run.answers {
		if !slices.Contains(words, settledAsk.Word) {
			continue
		}
		for _, answer := range settledAsk.Answers {
			for _, listedDocument := range answer.ListedDocuments {
				if run.partitions.PartitionOf(listedDocument.Hash) != partition {
					continue
				}
				documents.Add(listedDocument.Hash)
			}
		}
	}

	return documents
}

func (run *Run) Finish() Answers {
	close(run.asksToPut)
	for settledAsk := range run.settledAsksAsTheySettle {
		run.record(settledAsk)
	}

	return run.answers
}
