package wordjoined

func joinedDocumentsFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	crossCheckedDocumentsRound crossCheckedDocumentsRound,
) distinctDocuments {
	return matchedAndHeldDocumentsRound.documentsListedByPeersPerQueryWord().
		unitedWith(crossCheckedDocumentsRound.documentsFoundByCrossCheckingPerQueryWord()).
		documentsOfEveryQueryWord()
}
