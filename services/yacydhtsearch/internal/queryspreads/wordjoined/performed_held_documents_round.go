package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
)

type PerformedHeldDocumentsRound struct {
	AmountOfDocumentsPastTheHeldDocumentsCeiling      int
	AmountOfPeersAskedForHeldDocuments                int
	AmountOfPeersThatAnsweredHeldDocuments            int
	AmountOfEmptyHeldDocumentsAnswers                 int
	AmountOfJoinedDocumentsBeforeTheHeldDocumentsAsks int
	AmountOfJoinedDocuments                           int
}

func performedHeldDocumentsRoundFrom(
	round heldDocumentsRound,
	joinOfTheQuery joinOfTheQuery,
) PerformedHeldDocumentsRound {
	return PerformedHeldDocumentsRound{
		AmountOfDocumentsPastTheHeldDocumentsCeiling: round.
			amountOfDocumentsPastTheHeldDocumentsCeiling,
		AmountOfPeersAskedForHeldDocuments: amountOfPeersAcross(round.asks, peerOfHeldDocumentsAsk),
		AmountOfPeersThatAnsweredHeldDocuments: amountOfPeersAcross(
			round.answeredAsks, peerOfAnsweredHeldDocumentsAsk,
		),
		AmountOfEmptyHeldDocumentsAnswers: amountOfEmptyHeldDocumentsAnswers(round.answeredAsks),
		AmountOfJoinedDocumentsBeforeTheHeldDocumentsAsks: len(
			joinOfTheQuery.documentsJoinedBeforeTheHeldDocumentsAsks,
		),
		AmountOfJoinedDocuments: len(joinOfTheQuery.joinedDocuments),
	}
}

func peerOfHeldDocumentsAsk(ask peerasks.HeldDocumentsAsk) peerdirectory.AskablePeer {
	return ask.Peer
}

func peerOfAnsweredHeldDocumentsAsk(
	answeredAsk peerasks.AnsweredHeldDocumentsAsk,
) peerdirectory.AskablePeer {
	return answeredAsk.Ask.Peer
}

func amountOfEmptyHeldDocumentsAnswers(answeredAsks []peerasks.AnsweredHeldDocumentsAsk) int {
	amount := 0
	for _, answeredAsk := range answeredAsks {
		if len(answeredAsk.DocumentsHeldForTheWord) > 0 {
			continue
		}
		amount++
	}

	return amount
}
