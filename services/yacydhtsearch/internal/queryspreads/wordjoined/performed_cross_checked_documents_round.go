package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
)

type PerformedCrossCheckedDocumentsRound struct {
	AmountOfDocumentsPastTheCrossCheckedDocumentsCeiling int
	AmountOfPeersAskedForCrossCheckedDocuments           int
	AmountOfPeersThatAnsweredCrossCheckedDocuments       int
	AmountOfEmptyCrossCheckedDocumentsAnswers            int
	AmountOfDocumentsJoinedWithoutCrossChecking          int
	AmountOfDocumentsJoinedWithCrossChecking             int
}

func performedCrossCheckedDocumentsRoundFrom(
	round crossCheckedDocumentsRound,
	joinOfTheQuery joinOfTheQuery,
) PerformedCrossCheckedDocumentsRound {
	return PerformedCrossCheckedDocumentsRound{
		AmountOfDocumentsPastTheCrossCheckedDocumentsCeiling: round.
			amountOfDocumentsPastTheCrossCheckedDocumentsCeiling,
		AmountOfPeersAskedForCrossCheckedDocuments: amountOfPeersAcross(
			round.asks,
			peerOfCrossCheckedDocumentsAsk,
		),
		AmountOfPeersThatAnsweredCrossCheckedDocuments: amountOfPeersAcross(
			round.answeredAsks, peerOfAnsweredCrossCheckedDocumentsAsk,
		),
		AmountOfEmptyCrossCheckedDocumentsAnswers: amountOfEmptyCrossCheckedDocumentsAnswers(
			round.answeredAsks,
		),
		AmountOfDocumentsJoinedWithoutCrossChecking: len(
			joinOfTheQuery.documentsJoinedWithoutCrossChecking,
		),
		AmountOfDocumentsJoinedWithCrossChecking: len(
			joinOfTheQuery.documentsJoinedWithCrossChecking,
		),
	}
}

func peerOfCrossCheckedDocumentsAsk(
	ask peerasks.CrossCheckedDocumentsAsk,
) peerdirectory.AskablePeer {
	return ask.Peer
}

func peerOfAnsweredCrossCheckedDocumentsAsk(
	answeredAsk peerasks.AnsweredCrossCheckedDocumentsAsk,
) peerdirectory.AskablePeer {
	return answeredAsk.Ask.Peer
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
