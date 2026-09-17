package wordjoined

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type joinOfTheQuery struct {
	documentsJoinedBeforeTheCrossCheckedDocumentsAsks map[yacymodel.URLHash]struct{}
	joinedDocuments                                   map[yacymodel.URLHash]struct{}
}

func joinOfTheQueryFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	crossCheckedDocumentsRound crossCheckedDocumentsRound,
) joinOfTheQuery {
	documentsHeldPerQueryWordInTheSecondRound := crossCheckedDocumentsRound.documentsHeldPerQueryWord()
	documentsJoinedBeforeTheCrossCheckedDocumentsAsks := matchedAndHeldDocumentsRound.leadingQueryWord().
		documentsListed()
	joinedDocuments := maps.Clone(documentsJoinedBeforeTheCrossCheckedDocumentsAsks)
	for _, queryWord := range matchedAndHeldDocumentsRound.queryWordsBesideTheLeadingQueryWord() {
		documentsListedForTheQueryWord := queryWord.documentsListed()
		documentsJoinedBeforeTheCrossCheckedDocumentsAsks = documentsHeldInSomeRoundAmong(
			documentsJoinedBeforeTheCrossCheckedDocumentsAsks, documentsListedForTheQueryWord,
		)
		joinedDocuments = documentsHeldInSomeRoundAmong(
			joinedDocuments,
			documentsListedForTheQueryWord,
			documentsHeldPerQueryWordInTheSecondRound[queryWord.word],
		)
	}

	return joinOfTheQuery{
		documentsJoinedBeforeTheCrossCheckedDocumentsAsks: documentsJoinedBeforeTheCrossCheckedDocumentsAsks,
		joinedDocuments: joinedDocuments,
	}
}

func documentsHeldInSomeRoundAmong(
	documents map[yacymodel.URLHash]struct{},
	documentsHeldInEachRound ...map[yacymodel.URLHash]struct{},
) map[yacymodel.URLHash]struct{} {
	keptDocuments := make(map[yacymodel.URLHash]struct{}, len(documents))
	for document := range documents {
		for _, documentsHeldInOneRound := range documentsHeldInEachRound {
			if _, held := documentsHeldInOneRound[document]; held {
				keptDocuments[document] = struct{}{}

				break
			}
		}
	}

	return keptDocuments
}
