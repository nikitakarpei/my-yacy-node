package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
)

type PerformedCrossCheckedDocumentsRound struct {
	AmountOfDocumentsSentForCrossChecking                int
	AmountOfDocumentsPastTheCrossCheckedDocumentsCeiling int
	AmountOfEmptyCrossCheckedDocumentsAnswers            int
	AmountOfJoinedDocuments                              int
	AmountOfJoinedDocumentsFoundOnlyByCrossChecking      int
	PeerStandings                                        []peerjudgements.PeerStanding
	JudgedPeers                                          []peerjudgements.JudgedPeer
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
		PeerStandings: round.peerStandings,
		JudgedPeers:   round.judgedPeers,
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
