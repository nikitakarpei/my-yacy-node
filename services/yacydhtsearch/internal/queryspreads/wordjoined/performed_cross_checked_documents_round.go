package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

type PerformedCrossCheckedDocumentsRound struct {
	AmountOfDocumentsSentForCrossChecking                int
	AmountOfDocumentsPastTheCrossCheckedDocumentsCeiling int
	AmountOfEmptyCrossCheckedDocumentsAnswers            int
	AmountOfJoinedDocuments                              int
	AmountOfJoinedDocumentsFoundOnlyByCrossChecking      int
}

func performedCrossCheckedDocumentsRoundFrom(
	round crossCheckedDocumentsRound,
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	joinedDocuments distinctDocuments,
) PerformedCrossCheckedDocumentsRound {
	return PerformedCrossCheckedDocumentsRound{
		AmountOfDocumentsSentForCrossChecking: amountOfDocumentsSentForCrossCheckingAcross(
			round.asks,
		),
		AmountOfDocumentsPastTheCrossCheckedDocumentsCeiling: round.
			amountOfDocumentsPastTheCrossCheckedDocumentsCeiling,
		AmountOfEmptyCrossCheckedDocumentsAnswers: amountOfEmptyCrossCheckedDocumentsAnswers(
			round.answeredAsks,
		),
		AmountOfJoinedDocuments: len(joinedDocuments),
		AmountOfJoinedDocumentsFoundOnlyByCrossChecking: len(joinedDocuments) - len(
			matchedAndHeldDocumentsRound.documentsListedByPeersPerQueryWord().
				documentsOfEveryQueryWord(),
		),
	}
}

func amountOfDocumentsSentForCrossCheckingAcross(
	asks []peerasks.CrossCheckedDocumentsAsk,
) int {
	amount := 0
	for _, ask := range asks {
		amount += len(ask.Documents)
	}

	return amount
}

func amountOfEmptyCrossCheckedDocumentsAnswers(
	answeredAsks []peerasks.AnsweredCrossCheckedDocumentsAsk,
) int {
	amount := 0
	for _, answeredAsk := range answeredAsks {
		if len(answeredAsk.DocumentsHeldForTheWord) > 0 {
			continue
		}
		amount++
	}

	return amount
}
