package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordSplit struct {
	candidateWords []yacymodel.Hash
	otherWords     []yacymodel.Hash
}

func wordSplitBy(leadingQueryWord yacymodel.Hash, query searchquery.Query) wordSplit {
	split := wordSplit{
		candidateWords: []yacymodel.Hash{leadingQueryWord},
	}
	for _, queryWord := range query.WordHashes() {
		if queryWord == leadingQueryWord {
			continue
		}
		split.otherWords = append(split.otherWords, queryWord)
	}
	for _, compoundWord := range query.CompoundWords {
		if slices.Contains(compoundWord.WordHashes, leadingQueryWord) {
			split.candidateWords = append(split.candidateWords, compoundWord.Hash)

			continue
		}
		split.otherWords = append(split.otherWords, compoundWord.Hash)
	}

	return split
}
