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
	skipped    map[wordPartitionKey]struct{}
	answered   []wordpartitionasks.SettledAsk
	endedWords map[yacymodel.Hash]struct{}
}

func noAsksSentYet(partitions yacymodel.DHTRingPartitions) *sentAsks {
	return &sentAsks{
		partitions: partitions,
		sent:       map[wordPartitionKey]struct{}{},
		pending:    map[wordPartitionKey]struct{}{},
		skipped:    map[wordPartitionKey]struct{}{},
		endedWords: map[yacymodel.Hash]struct{}{},
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

func (asks *sentAsks) settle(settledAsk wordpartitionasks.SettledAsk) {
	delete(asks.pending, wordPartitionKeyOf(settledAsk.Ask))
	asks.answered = append(asks.answered, settledAsk)
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
	var documentsOfEachAnswer []yacymodel.URLHashes
	for _, answered := range asks.answered {
		if !slices.Contains(words, answered.Word) {
			continue
		}
		for _, answer := range answered.Answers {
			documentsOfEachAnswer = append(
				documentsOfEachAnswer,
				asks.partitions.URLHashesIn(partition, answer.ListedDocuments),
			)
		}
	}

	return yacymodel.UnionOf(documentsOfEachAnswer...)
}

func (asks *sentAsks) skip(words []yacymodel.Hash, partition uint) {
	for _, word := range words {
		asks.skipped[wordPartitionKey{word: word, partition: partition}] = struct{}{}
	}
}

func (asks *sentAsks) newlyEndedAmong(words []yacymodel.Hash) []yacymodel.Hash {
	var newlyEndedWords []yacymodel.Hash
	for _, word := range words {
		if _, ended := asks.endedWords[word]; ended || !asks.everyPartitionSettledFor(word) {
			continue
		}
		asks.endedWords[word] = struct{}{}
		newlyEndedWords = append(newlyEndedWords, word)
	}

	return newlyEndedWords
}

func (asks *sentAsks) everyPartitionSettledFor(word yacymodel.Hash) bool {
	for partition := range uint(asks.partitions) {
		key := wordPartitionKey{word: word, partition: partition}
		_, sent := asks.sent[key]
		_, skipped := asks.skipped[key]
		_, pending := asks.pending[key]
		if pending || !sent && !skipped {
			return false
		}
	}

	return true
}

func (asks *sentAsks) amountOfDocumentsListed() int {
	amount := 0
	for _, answered := range asks.answered {
		for _, answer := range answered.Answers {
			amount += answer.ListedDocuments.Amount()
		}
	}

	return amount
}
