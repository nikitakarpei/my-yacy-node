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
		AmountOfCrossCheckCandidatesNoPeerTook: amountOfCrossCheckCandidatesIn(
			matchedAndHeldDocumentsRound,
		) - amountOfDocumentsSentForCrossCheckingAcross(round.asks),
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

func amountOfCrossCheckCandidatesIn(matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound) int {
	documentsOfTheLeadingQueryWord := matchedAndHeldDocumentsRound.
		documentsOfTheLeadingQueryWordMostListedFirst()
	amount := 0
	for _, queryWord := range matchedAndHeldDocumentsRound.partlyListedQueryWordsBesideTheLeadingQueryWord() {
		amount += len(queryWord.documentsNotListedByItsPeersAmong(documentsOfTheLeadingQueryWord))
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
