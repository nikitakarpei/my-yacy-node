package documentamounts

import (
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Measurement struct {
	mutex               sync.Mutex
	queryWords          []yacymodel.Hash
	answersPerQueryWord map[yacymodel.Hash]answersOfWord
}

func NoneMeasuredYet(query searchquery.Query, partitions yacymodel.DHTRingPartitions) *Measurement {
	answersPerQueryWord := make(map[yacymodel.Hash]answersOfWord, len(query.WordHashes()))
	for _, word := range query.WordHashes() {
		answersPerQueryWord[word] = make(answersOfWord, partitions)
	}

	return &Measurement{queryWords: query.WordHashes(), answersPerQueryWord: answersPerQueryWord}
}

func (measurement *Measurement) WordPartitionAnswered(
	word yacymodel.Hash,
	partition uint,
	answers []wordpartitionasks.ReplicaAnswer,
) {
	measurement.mutex.Lock()
	defer measurement.mutex.Unlock()
	answersOfTheWord, queryWord := measurement.answersPerQueryWord[word]
	if !queryWord {
		return
	}
	answersOfTheWord[partition] = append(answersOfTheWord[partition], answers...)
}

func (measurement *Measurement) HeldPerQueryWord() map[yacymodel.Hash]int {
	measurement.mutex.Lock()
	defer measurement.mutex.Unlock()
	amountHeldPerQueryWord := make(map[yacymodel.Hash]int, len(measurement.queryWords))
	for word, answers := range measurement.answersPerQueryWord {
		if amountHeld, counted := answers.estimatedAmountHeld().Get(); counted {
			amountHeldPerQueryWord[word] = amountHeld
		}
	}

	return amountHeldPerQueryWord
}

func (measurement *Measurement) InAPartitionPerQueryWord() map[yacymodel.Hash]int {
	measurement.mutex.Lock()
	defer measurement.mutex.Unlock()
	amountInAPartitionPerQueryWord := make(map[yacymodel.Hash]int, len(measurement.queryWords))
	for word, answers := range measurement.answersPerQueryWord {
		if amountInAPartition, counted := answers.amountInAPartition().Get(); counted {
			amountInAPartitionPerQueryWord[word] = amountInAPartition
		}
	}

	return amountInAPartitionPerQueryWord
}

func (measurement *Measurement) AmountOfQueryWordsHeldByNoPeer() int {
	measurement.mutex.Lock()
	defer measurement.mutex.Unlock()
	amount := 0
	for _, answers := range measurement.answersPerQueryWord {
		if len(answers.documents()) == 0 {
			amount++
		}
	}

	return amount
}

func (measurement *Measurement) AmountListedOf(word yacymodel.Hash) int {
	measurement.mutex.Lock()
	defer measurement.mutex.Unlock()
	return len(measurement.answersPerQueryWord[word].documents())
}
