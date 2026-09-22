package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type crossCheckCandidatesOfQueryWord struct {
	queryWord queryWordAcrossReplicas
	documents []yacymodel.URLHash
}

func crossCheckCandidatesIn(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
) []crossCheckCandidatesOfQueryWord {
	documentsOfTheLeadingQueryWord := matchedAndHeldDocumentsRound.
		documentsOfTheLeadingQueryWordMostListedFirst()
	partlyListedQueryWords := matchedAndHeldDocumentsRound.partlyListedQueryWordsBesideTheLeadingQueryWord()
	candidates := make([]crossCheckCandidatesOfQueryWord, 0, len(partlyListedQueryWords))
	for _, queryWord := range partlyListedQueryWords {
		candidates = append(candidates, crossCheckCandidatesOfQueryWord{
			queryWord: queryWord,
			documents: queryWord.documentsNotListedByItsPeersAmong(documentsOfTheLeadingQueryWord),
		})
	}

	return candidates
}
