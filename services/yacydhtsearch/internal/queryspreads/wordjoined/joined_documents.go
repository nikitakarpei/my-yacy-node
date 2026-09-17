package wordjoined

import (
	"maps"
)

func joinedDocumentsFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	crossCheckedDocumentsRound crossCheckedDocumentsRound,
) distinctDocuments {
	return documentsFoundPerQueryWordFrom(
		matchedAndHeldDocumentsRound.documentsListedByPeersPerQueryWord(),
		crossCheckedDocumentsRound.documentsFoundByCrossCheckingPerQueryWord(),
	).documentsOfEveryQueryWord()
}

func documentsFoundPerQueryWordFrom(
	documentsListedByPeersPerQueryWord documentsPerQueryWord,
	documentsFoundByCrossCheckingPerQueryWord documentsPerQueryWord,
) documentsPerQueryWord {
	documentsFoundPerQueryWord := make(
		documentsPerQueryWord, len(documentsListedByPeersPerQueryWord),
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
