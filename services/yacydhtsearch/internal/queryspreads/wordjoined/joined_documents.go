package wordjoined

func joinedDocumentsFrom(
	abstractsRound abstractsRound,
	crossCheckedDocumentsRound crossCheckedDocumentsRound,
) distinctDocuments {
	return abstractsRound.documentsInTheAbstractsPerQueryWord().
		unitedWith(crossCheckedDocumentsRound.documentsFoundByCrossCheckingPerQueryWord()).
		documentsOfEveryQueryWord()
}
