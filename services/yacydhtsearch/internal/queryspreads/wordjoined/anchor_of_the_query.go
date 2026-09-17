package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type anchorOfTheQuery struct {
	word      yacymodel.Hash
	documents map[yacymodel.URLHash]struct{}
}

func anchorOfTheQueryFrom(
	queryWords []yacymodel.Hash,
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
	amountOfDocumentsHeldPerQueryWord map[yacymodel.Hash]int,
) anchorOfTheQuery {
	anchorWord := anchorWordOf(queryWords, amountOfDocumentsHeldPerQueryWord)

	return anchorOfTheQuery{
		word:      anchorWord,
		documents: documentsHeldForTheWordAcrossAnswers(answeredAsks, anchorWord),
	}
}

func anchorWordOf(
	queryWords []yacymodel.Hash,
	amountOfDocumentsHeldPerQueryWord map[yacymodel.Hash]int,
) yacymodel.Hash {
	if word, counted := queryWordWithTheFewestDocumentsAmong(
		queryWords, amountOfDocumentsHeldPerQueryWord,
	); counted {
		return word
	}

	return queryWords[0]
}

func queryWordWithTheFewestDocumentsAmong(
	queryWords []yacymodel.Hash,
	amountOfDocumentsHeldPerQueryWord map[yacymodel.Hash]int,
) (yacymodel.Hash, bool) {
	var wordWithTheFewestDocuments yacymodel.Hash
	fewestDocuments := 0
	counted := false
	for _, queryWord := range queryWords {
		amountOfDocumentsHeld, countedForTheWord := amountOfDocumentsHeldPerQueryWord[queryWord]
		if !countedForTheWord || (counted && amountOfDocumentsHeld >= fewestDocuments) {
			continue
		}
		wordWithTheFewestDocuments = queryWord
		fewestDocuments = amountOfDocumentsHeld
		counted = true
	}

	return wordWithTheFewestDocuments, counted
}

func documentsHeldForTheWordAcrossAnswers(
	answeredAsks []peerasks.AnsweredMatchedAndHeldDocumentsAsk,
	word yacymodel.Hash,
) map[yacymodel.URLHash]struct{} {
	documentsHeldForTheWord := map[yacymodel.URLHash]struct{}{}
	for _, answeredAsk := range answeredAsks {
		if answeredAsk.Ask.Word != word {
			continue
		}
		for _, document := range answeredAsk.DocumentsHeldForTheWord {
			documentsHeldForTheWord[document] = struct{}{}
		}
	}

	return documentsHeldForTheWord
}
