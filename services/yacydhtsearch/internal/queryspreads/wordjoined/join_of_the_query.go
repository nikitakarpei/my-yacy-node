package wordjoined

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type joinOfTheQuery struct {
	documentsJoinedWithoutCrossChecking map[yacymodel.URLHash]struct{}
	documentsJoinedWithCrossChecking    map[yacymodel.URLHash]struct{}
}

func joinOfTheQueryFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	crossCheckedDocumentsRound crossCheckedDocumentsRound,
) joinOfTheQuery {
	return joinOfTheQuery{
		documentsJoinedWithoutCrossChecking: documentsJoinedFrom(matchedAndHeldDocumentsRound, nil),
		documentsJoinedWithCrossChecking: documentsJoinedFrom(
			matchedAndHeldDocumentsRound,
			crossCheckedDocumentsRound.documentsFoundByCrossCheckingPerQueryWord(),
		),
	}
}

func documentsJoinedFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	documentsFoundByCrossCheckingPerQueryWord map[yacymodel.Hash]map[yacymodel.URLHash]struct{},
) map[yacymodel.URLHash]struct{} {
	documentsJoined := matchedAndHeldDocumentsRound.leadingQueryWord().documentsListedByPeers()
	for _, queryWord := range matchedAndHeldDocumentsRound.queryWordsBesideTheLeadingQueryWord() {
		documentsJoined = documentsAlsoAmong(
			documentsJoined,
			documentsFoundForTheQueryWordFrom(
				queryWord.documentsListedByPeers(),
				documentsFoundByCrossCheckingPerQueryWord[queryWord.word],
			),
		)
	}

	return documentsJoined
}

func documentsAlsoAmong(
	documentsJoined map[yacymodel.URLHash]struct{},
	documentsFoundForTheQueryWord map[yacymodel.URLHash]struct{},
) map[yacymodel.URLHash]struct{} {
	keptDocuments := make(map[yacymodel.URLHash]struct{}, len(documentsJoined))
	for document := range documentsJoined {
		if _, found := documentsFoundForTheQueryWord[document]; !found {
			continue
		}
		keptDocuments[document] = struct{}{}
	}

	return keptDocuments
}

func documentsFoundForTheQueryWordFrom(
	documentsListedByPeers map[yacymodel.URLHash]struct{},
	documentsFoundByCrossChecking map[yacymodel.URLHash]struct{},
) map[yacymodel.URLHash]struct{} {
	documentsFoundForTheQueryWord := maps.Clone(documentsListedByPeers)
	maps.Copy(documentsFoundForTheQueryWord, documentsFoundByCrossChecking)

	return documentsFoundForTheQueryWord
}
