package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
)

type PerformedCrossCheckedDocumentsRound struct {
	AmountOfDocumentsSentForCrossChecking           int
	AmountOfCrossCheckCandidatesNoPeerTook          int
	AmountOfEmptyCrossCheckedDocumentsAnswers       int
	AmountOfJoinedDocuments                         int
	AmountOfJoinedDocumentsFoundOnlyByCrossChecking int
	PeerStandings                                   []peerjudgements.PeerStanding
	JudgedPeers                                     []peerjudgements.JudgedPeer
}

func performedCrossCheckedDocumentsRoundFrom(
	round crossCheckedDocumentsRound,
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	peerStandings peerjudgements.PeerStandings,
	judgedPeers []peerjudgements.JudgedPeer,
	joinedDocuments distinctDocuments,
) PerformedCrossCheckedDocumentsRound {
	return PerformedCrossCheckedDocumentsRound{
		AmountOfDocumentsSentForCrossChecking: amountOfDocumentsSentForCrossCheckingAcross(
			round.asks,
		),
		AmountOfCrossCheckCandidatesNoPeerTook: amountOfCrossCheckCandidatesAcross(
			round.candidates,
		) -
			amountOfDocumentsSentForCrossCheckingAcross(
				round.asks,
			),
		AmountOfEmptyCrossCheckedDocumentsAnswers: amountOfEmptyCrossCheckedDocumentsAnswers(
			round.answeredAsks,
		),
		AmountOfJoinedDocuments: len(joinedDocuments),
		AmountOfJoinedDocumentsFoundOnlyByCrossChecking: amountOfJoinedDocumentsFoundOnlyByCrossChecking(
			joinedDocuments,
			matchedAndHeldDocumentsRound,
		),
		PeerStandings: peerStandings,
		JudgedPeers:   judgedPeers,
	}
}

func amountOfCrossCheckCandidatesAcross(candidates []crossCheckCandidatesOfWordPartition) int {
	amount := 0
	for _, candidatesOfWordPartition := range candidates {
		amount += len(candidatesOfWordPartition.documents)
	}

	return amount
}

func amountOfJoinedDocumentsFoundOnlyByCrossChecking(
	joinedDocuments distinctDocuments,
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
) int {
	documentsOfEveryQueryWordListedByPeers := matchedAndHeldDocumentsRound.
		documentsListedByPeersPerQueryWord().documentsOfEveryQueryWord()

	return len(joinedDocuments) - len(documentsOfEveryQueryWordListedByPeers)
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
		if len(answeredAsk.DocumentsListedForTheWord) > 0 {
			continue
		}
		amount++
	}

	return amount
}
