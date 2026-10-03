package documentasks

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type sentAsks struct {
	partitions yacymodel.DHTRingPartitions
	sent       map[wordPartitionKey]struct{}
	pending    map[wordPartitionKey]struct{}
	settled    []wordpartitionasks.SettledAsk
}

func noAsksSentYet(partitions yacymodel.DHTRingPartitions) *sentAsks {
	return &sentAsks{
		partitions: partitions,
		sent:       map[wordPartitionKey]struct{}{},
		pending:    map[wordPartitionKey]struct{}{},
	}
}

func (asks *sentAsks) add(sentAsks []wordpartitionasks.Ask) {
	for _, ask := range sentAsks {
		asks.sent[wordPartitionKeyOf(ask)] = struct{}{}
		asks.pending[wordPartitionKeyOf(ask)] = struct{}{}
	}
}

func (asks *sentAsks) contain(ask wordpartitionasks.Ask) bool {
	_, sent := asks.sent[wordPartitionKeyOf(ask)]

	return sent
}

func (asks *sentAsks) containEvery(candidates []wordpartitionasks.Ask) bool {
	return !slices.ContainsFunc(candidates, func(ask wordpartitionasks.Ask) bool {
		return !asks.contain(ask)
	})
}

func (asks *sentAsks) settle(settledAsk wordpartitionasks.SettledAsk) {
	asks.settled = append(asks.settled, settledAsk)
	delete(asks.pending, wordPartitionKeyOf(settledAsk.Ask))
}

func (asks *sentAsks) nonePendingFor(words []yacymodel.Hash) bool {
	for key := range asks.pending {
		if slices.Contains(words, key.word) {
			return false
		}
	}

	return true
}

func (asks *sentAsks) nonePendingIn(partition uint, words []yacymodel.Hash) bool {
	for _, word := range words {
		if _, pending := asks.pending[wordPartitionKey{word: word, partition: partition}]; pending {
			return false
		}
	}

	return true
}

func (asks *sentAsks) documentsListedIn(
	partition uint,
	words []yacymodel.Hash,
) yacymodel.URLHashes {
	documents := yacymodel.URLHashes{}
	for _, settledAsk := range asks.settled {
		if !slices.Contains(words, settledAsk.Word) {
			continue
		}
		for _, answer := range settledAsk.Answers {
			for _, listedDocument := range answer.ListedDocuments {
				if asks.partitions.PartitionOf(listedDocument.Hash) != partition {
					continue
				}
				documents.Add(listedDocument.Hash)
			}
		}
	}

	return documents
}

func (asks *sentAsks) settledFor(words []yacymodel.Hash) Answers {
	return asks.settledWhere(func(settledAsk wordpartitionasks.SettledAsk) bool {
		return slices.Contains(words, settledAsk.Word)
	})
}

func (asks *sentAsks) settledIn(partition uint, words []yacymodel.Hash) Answers {
	return asks.settledWhere(func(settledAsk wordpartitionasks.SettledAsk) bool {
		return settledAsk.Partition == partition && slices.Contains(words, settledAsk.Word)
	})
}

func (asks *sentAsks) settledWhere(keep func(wordpartitionasks.SettledAsk) bool) Answers {
	return Answers{
		SettledAsks: slices.DeleteFunc(
			slices.Clone(asks.settled),
			func(settledAsk wordpartitionasks.SettledAsk) bool { return !keep(settledAsk) },
		),
		Partitions: asks.partitions,
	}
}
