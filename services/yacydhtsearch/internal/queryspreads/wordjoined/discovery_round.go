package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
)

type discoveryRound struct {
	answers                        *discoveryAnswers
	queryWordsFewestDocumentsFirst []wordAcrossReplicas
	compoundWords                  []compoundWordAcrossReplicas
	sampledPartition               uint
	amountOfQueryWordsWithASample  int
	chosenLeadingQueryWord         chosenLeadingQueryWord
	otherWordAsksPerPartition      map[uint]OtherWordAsks
}

func (round discoveryRound) leadingQueryWord() wordAcrossReplicas {
	chosenWord, chosen := round.chosenLeadingQueryWord.word.Get()
	if !chosen {
		return round.queryWordsFewestDocumentsFirst[0]
	}
	place := slices.IndexFunc(
		round.queryWordsFewestDocumentsFirst,
		func(queryWord wordAcrossReplicas) bool {
			return queryWord.hash == chosenWord
		},
	)

	return round.queryWordsFewestDocumentsFirst[place]
}

func (round discoveryRound) everyAnswer() []wordpartitionasks.ReplicaAnswer {
	return round.answers.everyAnswer()
}
