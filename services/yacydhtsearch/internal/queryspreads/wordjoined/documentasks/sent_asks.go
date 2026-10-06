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
	answered   []wordpartitionasks.SettledAsk
	answeredAt map[wordPartitionKey]int
}

func noAsksSentYet(partitions yacymodel.DHTRingPartitions) *sentAsks {
	return &sentAsks{
		partitions: partitions,
		sent:       map[wordPartitionKey]struct{}{},
		pending:    map[wordPartitionKey]struct{}{},
		answeredAt: map[wordPartitionKey]int{},
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
	delete(asks.pending, wordPartitionKeyOf(settledAsk.Ask))
	asks.answeredAt[wordPartitionKeyOf(settledAsk.Ask)] = len(asks.answered)
	asks.answered = append(asks.answered, settledAsk)
}

func (asks *sentAsks) answersOf(
	partition uint,
	words []yacymodel.Hash,
) []wordpartitionasks.SettledAsk {
	var answered []wordpartitionasks.SettledAsk
	for _, word := range words {
		place, settled := asks.answeredAt[wordPartitionKey{word: word, partition: partition}]
		if !settled {
			continue
		}
		answered = append(answered, asks.answered[place])
	}

	return answered
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
	for _, answered := range asks.answered {
		if !slices.Contains(words, answered.Word) {
			continue
		}
		for _, answer := range answered.Answers {
			for _, listedDocument := range answer.ListedDocuments {
				if asks.partitions.PartitionOf(listedDocument.Hash) == partition {
					documents.Add(listedDocument.Hash)
				}
			}
		}
	}

	return documents
}
