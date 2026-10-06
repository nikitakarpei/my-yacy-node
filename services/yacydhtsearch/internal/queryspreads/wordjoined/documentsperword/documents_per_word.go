// Package documentsperword tells, for each query word, what its word partitions
// answered: the amount of documents held, the amount in a partition, the
// complete abstracts of a partition, and the documents of the leading word.
package documentsperword

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type DocumentsPerWord struct {
	queryWords            []documentsOfWord
	answeredCompoundWords []searchquery.CompoundWord
}

func From(
	query searchquery.Query,
	answeredWordPartitions []documentasks.AnsweredWordPartition,
	partitions yacymodel.DHTRingPartitions,
) DocumentsPerWord {
	return DocumentsPerWord{
		queryWords: documentsOfEachWordFrom(query.WordHashes(), answeredWordPartitions, partitions),
		answeredCompoundWords: compoundWordsAnsweredAmong(
			query.CompoundWords, answeredWordPartitions,
		),
	}
}

func compoundWordsAnsweredAmong(
	compoundWords []searchquery.CompoundWord,
	answeredWordPartitions []documentasks.AnsweredWordPartition,
) []searchquery.CompoundWord {
	var answeredCompoundWords []searchquery.CompoundWord
	for _, compoundWord := range compoundWords {
		if !slices.ContainsFunc(
			answeredWordPartitions,
			func(answered documentasks.AnsweredWordPartition) bool {
				return answered.Word == compoundWord.Hash()
			},
		) {
			continue
		}
		answeredCompoundWords = append(answeredCompoundWords, compoundWord)
	}

	return answeredCompoundWords
}

func (documentsPerWord DocumentsPerWord) AmountHeldPerQueryWord() map[yacymodel.Hash]int {
	amountOfDocumentsHeldPerQueryWord := make(
		map[yacymodel.Hash]int,
		len(documentsPerWord.queryWords),
	)
	for _, queryWord := range documentsPerWord.queryWords {
		amountOfDocumentsHeld, counted := queryWord.estimatedAmountOfDocumentsHeld().Get()
		if !counted {
			continue
		}
		amountOfDocumentsHeldPerQueryWord[queryWord.word] = amountOfDocumentsHeld
	}

	return amountOfDocumentsHeldPerQueryWord
}

func (documentsPerWord DocumentsPerWord) AmountInAPartitionPerQueryWord() map[yacymodel.Hash]int {
	amountOfDocumentsPerQueryWord := make(map[yacymodel.Hash]int, len(documentsPerWord.queryWords))
	for _, queryWord := range documentsPerWord.queryWords {
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

func (documentsPerWord DocumentsPerWord) CompleteAbstractsIn(partition uint) []CompleteAbstract {
	var completeAbstracts []CompleteAbstract
	for _, queryWord := range documentsPerWord.queryWords {
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

func (documentsPerWord DocumentsPerWord) OfTheLeadingWord(
	leadingWord yacymodel.Optional[yacymodel.Hash],
) yacymodel.URLHashes {
	word, led := leadingWord.Get()
	if !led {
		return slices.MinFunc(documentsPerWord.queryWords, fewestDocumentsFirst).documents()
	}
	place := slices.IndexFunc(
		documentsPerWord.queryWords,
		func(queryWord documentsOfWord) bool { return queryWord.word == word },
	)

	return documentsPerWord.queryWords[place].documents()
}

func (documentsPerWord DocumentsPerWord) AmountOfQueryWords() int {
	return len(documentsPerWord.queryWords)
}

func (documentsPerWord DocumentsPerWord) AmountOfCompoundWords() int {
	return len(documentsPerWord.answeredCompoundWords)
}

func (documentsPerWord DocumentsPerWord) AmountOfQueryWordsHeldByNoPeer() int {
	amount := 0
	for _, queryWord := range documentsPerWord.queryWords {
		if len(queryWord.documents()) > 0 {
			continue
		}
		amount++
	}

	return amount
}
