package wordjoined

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type joinOfTheQuery struct {
	documentsJoinedBeforeTheHeldDocumentsAsks map[yacymodel.URLHash]struct{}
	joinedDocuments                           map[yacymodel.URLHash]struct{}
}

func joinOfTheQueryFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	heldDocumentsRound heldDocumentsRound,
) joinOfTheQuery {
	documentsHeldPerQueryWordInTheSecondRound := heldDocumentsRound.documentsHeldPerQueryWord()
	documentsJoinedBeforeTheHeldDocumentsAsks := matchedAndHeldDocumentsRound.anchor().
		documentsHeld()
	joinedDocuments := maps.Clone(documentsJoinedBeforeTheHeldDocumentsAsks)
	for _, queryWord := range matchedAndHeldDocumentsRound.queryWordsBesideTheAnchor() {
		documentsHeldInTheFirstRound := queryWord.documentsHeld()
		documentsJoinedBeforeTheHeldDocumentsAsks = documentsHeldInSomeRoundAmong(
			documentsJoinedBeforeTheHeldDocumentsAsks, documentsHeldInTheFirstRound,
		)
		joinedDocuments = documentsHeldInSomeRoundAmong(
			joinedDocuments,
			documentsHeldInTheFirstRound,
			documentsHeldPerQueryWordInTheSecondRound[queryWord.word],
		)
	}

	return joinOfTheQuery{
		documentsJoinedBeforeTheHeldDocumentsAsks: documentsJoinedBeforeTheHeldDocumentsAsks,
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
