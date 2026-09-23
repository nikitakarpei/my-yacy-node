package wordjoined

func joinedDocumentsFrom(
	abstractsRound abstractsRound,
	crossCheckedDocumentsRound crossCheckedDocumentsRound,
) distinctDocuments {
	return abstractsRound.documentsListedByPeersPerQueryWord().
		unitedWith(crossCheckedDocumentsRound.documentsFoundByCrossCheckingPerQueryWord()).
		documentsOfEveryQueryWord()
}
