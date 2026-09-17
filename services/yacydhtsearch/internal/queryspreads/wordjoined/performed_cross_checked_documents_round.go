package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedCrossCheckedDocumentsRound struct {
	AmountOfDocumentsPastTheCrossCheckedDocumentsCeiling int
	AmountOfPeersAskedForCrossCheckedDocuments           int
	AmountOfPeersThatAnsweredCrossCheckedDocuments       int
	AmountOfEmptyCrossCheckedDocumentsAnswers            int
	AmountOfJoinedDocuments                              int
	AmountOfJoinedDocumentsFoundOnlyByCrossChecking      int
}

func performedCrossCheckedDocumentsRoundFrom(
	round crossCheckedDocumentsRound,
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	joinedDocuments map[yacymodel.URLHash]struct{},
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
		AmountOfJoinedDocuments: len(joinedDocuments),
		AmountOfJoinedDocumentsFoundOnlyByCrossChecking: amountOfJoinedDocumentsFoundOnlyByCrossCheckingAmong(
			joinedDocuments,
			matchedAndHeldDocumentsRound.documentsListedByPeersPerQueryWord(),
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

func amountOfJoinedDocumentsFoundOnlyByCrossCheckingAmong(
	joinedDocuments map[yacymodel.URLHash]struct{},
	documentsListedByPeersPerQueryWord map[yacymodel.Hash]map[yacymodel.URLHash]struct{},
) int {
	amount := 0
	for document := range joinedDocuments {
		if isFoundForEveryQueryWord(document, documentsListedByPeersPerQueryWord) {
			continue
		}
		amount++
	}

	return amount
}
