package wordjoined

func joinedDocumentsFrom(
	abstractsRound abstractsRound,
	crossCheckRound crossCheckRound,
) distinctDocuments {
	return abstractsRound.documentsInTheAbstractsPerQueryWord().
		unitedWith(crossCheckRound.documentsFoundByCrossCheckingPerQueryWord()).
		documentsOfEveryQueryWord()
}
