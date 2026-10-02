// Package wordholdings holds what the replicas hold for each query word, and
// the documents that every query word holds.
package wordholdings

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Holdings struct {
	queryWords    []holdingsOfWord
	compoundWords []holdingsOfCompoundWord
}

func OfEachQueryWord(
	query searchquery.Query,
	settledAsks []wordpartitionasks.SettledAsk,
	partitions yacymodel.DHTRingPartitions,
) Holdings {
	return Holdings{
		queryWords: holdingsOfEachWordFrom(query.WordHashes(), settledAsks, partitions),
		compoundWords: holdingsOfEachCompoundWordFrom(
			query.CompoundWords, settledAsks, partitions,
		),
	}
}

func (holdings Holdings) DocumentsOfEveryWord() yacymodel.URLHashes {
	return holdings.documentsPerQueryWord().documentsOfEveryQueryWord()
}

func (holdings Holdings) documentsPerQueryWord() documentsPerQueryWord {
	documentsOfEachQueryWord := make(documentsPerQueryWord, len(holdings.queryWords))
	for _, queryWord := range holdings.queryWords {
		documentsOfEachQueryWord[queryWord.word] = queryWord.documents()
	}
	for _, compoundWord := range holdings.compoundWords {
		for _, word := range compoundWord.PartHashes() {
			documentsOfEachQueryWord.add(word, compoundWord.documents())
		}
	}

	return documentsOfEachQueryWord
}

func (holdings Holdings) AmountOfDocumentsHeldPerQueryWord() map[yacymodel.Hash]int {
	amountOfDocumentsHeldPerQueryWord := make(map[yacymodel.Hash]int, len(holdings.queryWords))
	for _, queryWord := range holdings.queryWords {
		amountOfDocumentsHeld, counted := queryWord.estimatedAmountOfDocumentsHeld().Get()
		if !counted {
			continue
		}
		amountOfDocumentsHeldPerQueryWord[queryWord.word] = amountOfDocumentsHeld
	}

	return amountOfDocumentsHeldPerQueryWord
}

func (holdings Holdings) AmountOfDocumentsInAPartitionPerQueryWord() map[yacymodel.Hash]int {
	amountOfDocumentsPerQueryWord := make(map[yacymodel.Hash]int, len(holdings.queryWords))
	for _, queryWord := range holdings.queryWords {
		amountOfDocuments, counted := queryWord.amountOfDocumentsInAPartition().Get()
		if !counted {
			continue
		}
		amountOfDocumentsPerQueryWord[queryWord.word] = amountOfDocuments
	}

	return amountOfDocumentsPerQueryWord
}

type CompleteAbstract struct {
	Word                    yacymodel.Hash
	DocumentsInThePartition yacymodel.URLHashes
}

func (holdings Holdings) CompleteAbstractsIn(partition uint) []CompleteAbstract {
	var completeAbstracts []CompleteAbstract
	for _, queryWord := range holdings.queryWords {
		documentsInThePartition, complete := queryWord.completeAbstractIn(partition).Get()
		if !complete {
			continue
		}
		completeAbstracts = append(completeAbstracts, CompleteAbstract{
			Word:                    queryWord.word,
			DocumentsInThePartition: documentsInThePartition,
		})
	}

	return completeAbstracts
}

func (holdings Holdings) DocumentsOfTheLeadingWord(
	leadingWord yacymodel.Optional[yacymodel.Hash],
) yacymodel.URLHashes {
	word, led := leadingWord.Get()
	if !led {
		return slices.MinFunc(holdings.queryWords, fewestDocumentsFirst).documents()
	}
	place := slices.IndexFunc(
		holdings.queryWords,
		func(queryWord holdingsOfWord) bool { return queryWord.word == word },
	)

	return holdings.queryWords[place].documents()
}

func (holdings Holdings) AmountOfQueryWords() int {
	return len(holdings.queryWords)
}

func (holdings Holdings) AmountOfCompoundWords() int {
	return len(holdings.compoundWords)
}

func (holdings Holdings) AmountOfQueryWordsHeldByNoPeer() int {
	amount := 0
	for _, queryWord := range holdings.queryWords {
		if len(queryWord.documents()) > 0 {
			continue
		}
		amount++
	}

	return amount
}
