// Package documentsperword holds, for each query word, the documents and counts
// the partitions answered with, and the documents every query word has.
package documentsperword

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type DocumentsPerWord struct {
	queryWords    []documentsOfWord
	compoundWords []documentsOfCompoundWord
}

func From(query searchquery.Query, documentAnswers documentasks.Answers) DocumentsPerWord {
	return DocumentsPerWord{
		queryWords: documentsOfEachWordFrom(
			query.WordHashes(), documentAnswers.SettledAsks, documentAnswers.Partitions,
		),
		compoundWords: documentsOfEachCompoundWordFrom(
			query.CompoundWords, documentAnswers.SettledAsks, documentAnswers.Partitions,
		),
	}
}

func (documentsPerWord DocumentsPerWord) WithEveryWord() yacymodel.URLHashes {
	return documentsPerWord.documentsCountingForEachQueryWord().documentsWithEveryQueryWord()
}

func (documentsPerWord DocumentsPerWord) documentsCountingForEachQueryWord() documentsCountingForQueryWord {
	documentsOfEachQueryWord := make(
		documentsCountingForQueryWord,
		len(documentsPerWord.queryWords),
	)
	for _, queryWord := range documentsPerWord.queryWords {
		documentsOfEachQueryWord[queryWord.word] = queryWord.documents()
	}
	for _, compoundWord := range documentsPerWord.compoundWords {
		for _, word := range compoundWord.PartHashes() {
			documentsOfEachQueryWord.add(word, compoundWord.documents())
		}
	}

	return documentsOfEachQueryWord
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
	return len(documentsPerWord.compoundWords)
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
