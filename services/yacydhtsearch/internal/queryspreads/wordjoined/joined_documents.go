package wordjoined

func joinedDocumentsFrom(
	discoveryRound discoveryRound,
	crossCheckRound crossCheckRound,
) distinctDocuments {
	return discoveryRound.documentsPerQueryWord().
		unitedWith(crossCheckRound.documentsPerQueryWord()).
		documentsOfEveryQueryWord()
}
