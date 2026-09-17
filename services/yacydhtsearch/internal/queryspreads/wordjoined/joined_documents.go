package wordjoined

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func joinedDocumentsFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	crossCheckedDocumentsRound crossCheckedDocumentsRound,
) map[yacymodel.URLHash]struct{} {
	return documentsFoundForEveryQueryWordFrom(
		documentsFoundPerQueryWordFrom(
			matchedAndHeldDocumentsRound.documentsListedByPeersPerQueryWord(),
			crossCheckedDocumentsRound.documentsFoundByCrossCheckingPerQueryWord(),
		),
	)
}

func documentsFoundPerQueryWordFrom(
	documentsListedByPeersPerQueryWord map[yacymodel.Hash]map[yacymodel.URLHash]struct{},
	documentsFoundByCrossCheckingPerQueryWord map[yacymodel.Hash]map[yacymodel.URLHash]struct{},
) map[yacymodel.Hash]map[yacymodel.URLHash]struct{} {
	documentsFoundPerQueryWord := make(
		map[yacymodel.Hash]map[yacymodel.URLHash]struct{}, len(documentsListedByPeersPerQueryWord),
	)
	for queryWord, documentsListedByPeers := range documentsListedByPeersPerQueryWord {
		documentsFoundForTheQueryWord := maps.Clone(documentsListedByPeers)
		maps.Copy(
			documentsFoundForTheQueryWord,
			documentsFoundByCrossCheckingPerQueryWord[queryWord],
		)
		documentsFoundPerQueryWord[queryWord] = documentsFoundForTheQueryWord
	}

	return documentsFoundPerQueryWord
}

func documentsFoundForEveryQueryWordFrom(
	documentsFoundPerQueryWord map[yacymodel.Hash]map[yacymodel.URLHash]struct{},
) map[yacymodel.URLHash]struct{} {
	documentsFoundForEveryQueryWord := map[yacymodel.URLHash]struct{}{}
	for _, documentsFoundForOneQueryWord := range documentsFoundPerQueryWord {
		for document := range documentsFoundForOneQueryWord {
			if !isFoundForEveryQueryWord(document, documentsFoundPerQueryWord) {
				continue
			}
			documentsFoundForEveryQueryWord[document] = struct{}{}
		}
	}

	return documentsFoundForEveryQueryWord
}

func isFoundForEveryQueryWord(
	document yacymodel.URLHash,
	documentsFoundPerQueryWord map[yacymodel.Hash]map[yacymodel.URLHash]struct{},
) bool {
	for _, documentsFoundForOneQueryWord := range documentsFoundPerQueryWord {
		if _, found := documentsFoundForOneQueryWord[document]; !found {
			return false
		}
	}

	return true
}
