package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type discoveryRound struct {
	queryWords                     []yacymodel.Hash
	answeredAsks                   []peerasks.AnsweredSearchDocumentsAsk
	queryWordsFewestDocumentsFirst []queryWordAcrossReplicas
	compoundWords                  []compoundWordAcrossReplicas
	holdersPerDocument             holdersPerDocument
	sampledPartition               uint
	amountOfQueryWordsWithASample  int
	leadingQueryWordFromTheSample  yacymodel.Optional[yacymodel.Hash]
	otherWordAsksPerPartition      map[uint]OtherWordAsks
}

func (round discoveryRound) leadingQueryWord() queryWordAcrossReplicas {
	leadingQueryWordFromTheSample, chosen := round.leadingQueryWordFromTheSample.Get()
	if !chosen {
		return round.queryWordsFewestDocumentsFirst[0]
	}
	place := slices.IndexFunc(
		round.queryWordsFewestDocumentsFirst,
		func(queryWord queryWordAcrossReplicas) bool {
			return queryWord.word == leadingQueryWordFromTheSample
		},
	)

	return round.queryWordsFewestDocumentsFirst[place]
}

func (round discoveryRound) amountOfDocumentsHeldPerQueryWord() map[yacymodel.Hash]int {
	amountOfDocumentsHeldPerQueryWord := make(
		map[yacymodel.Hash]int, len(round.queryWordsFewestDocumentsFirst),
	)
	for _, queryWord := range round.queryWordsFewestDocumentsFirst {
		amountOfDocumentsHeld, counted := queryWord.estimatedAmountOfDocumentsHeld().Get()
		if !counted {
			continue
		}
		amountOfDocumentsHeldPerQueryWord[queryWord.word] = amountOfDocumentsHeld
	}

	return amountOfDocumentsHeldPerQueryWord
}

func (round discoveryRound) joinedDocuments() distinctDocuments {
	return round.documentsPerQueryWord().documentsOfEveryQueryWord()
}

func (round discoveryRound) documentsPerQueryWord() documentsPerQueryWord {
	documentsOfEachQueryWord := make(
		documentsPerQueryWord,
		len(round.queryWordsFewestDocumentsFirst),
	)
	for _, queryWord := range round.queryWordsFewestDocumentsFirst {
		documentsOfEachQueryWord[queryWord.word] = queryWord.documents()
	}
	for _, compoundWord := range round.compoundWords {
		for _, word := range compoundWord.WordHashes {
			documentsOfEachQueryWord.add(word, compoundWord.documents())
		}
	}

	return documentsOfEachQueryWord
}
