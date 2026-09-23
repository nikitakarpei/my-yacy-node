package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedCrossCheckRound struct {
	AmountOfDocumentsSentForCrossChecking              int
	AmountOfCrossCheckCandidatesNoPeerTook             int
	AmountOfCrossCheckCandidatesRuledOutByAFullListing int
	AmountOfEmptyCrossCheckAnswers                     int
	AmountOfJoinedDocuments                            int
	AmountOfJoinedDocumentsFoundOnlyByCrossChecking    int
	JudgedPeers                                        []peerjudgements.JudgedPeer
}

func performedCrossCheckRoundFrom(
	round crossCheckRound,
	discoveryRound discoveryRound,
	judgedPeers []peerjudgements.JudgedPeer,
	joinedDocuments distinctDocuments,
) PerformedCrossCheckRound {
	return PerformedCrossCheckRound{
		AmountOfDocumentsSentForCrossChecking: amountOfDocumentsSentForCrossCheckingAcross(
			round.asks,
		),
		AmountOfCrossCheckCandidatesNoPeerTook: amountOfCrossCheckCandidatesAcross(
			round.candidates.ofWordPartitionsWithoutACompleteAbstract,
		) -
			amountOfDocumentsSentForCrossCheckingAcross(
				round.asks,
			),
		AmountOfCrossCheckCandidatesRuledOutByAFullListing: amountOfCrossCheckCandidatesAcross(
			round.candidates.ofWordPartitionsWithACompleteAbstract,
		),
		AmountOfEmptyCrossCheckAnswers: amountOfEmptyCrossCheckAnswers(
			round.answeredAsks,
		),
		AmountOfJoinedDocuments: len(joinedDocuments),
		AmountOfJoinedDocumentsFoundOnlyByCrossChecking: amountOfJoinedDocumentsFoundOnlyByCrossChecking(
			joinedDocuments,
			discoveryRound,
		),
		JudgedPeers: judgedPeers,
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
	discoveryRound discoveryRound,
) int {
	documentsOfEveryQueryWord := discoveryRound.
		documentsPerQueryWord().documentsOfEveryQueryWord()

	return len(joinedDocuments) - len(documentsOfEveryQueryWord)
}

type documentOfWord struct {
	word     yacymodel.Hash
	document yacymodel.URLHash
}

func amountOfDocumentsSentForCrossCheckingAcross(
	asks []peerasks.SearchDocumentsAsk,
) int {
	documentsSent := map[documentOfWord]struct{}{}
	for _, ask := range asks {
		for _, document := range ask.DocumentsToMatch {
			documentsSent[documentOfWord{word: ask.Word, document: document}] = struct{}{}
		}
	}

	return len(documentsSent)
}

func amountOfEmptyCrossCheckAnswers(
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
) int {
	amount := 0
	for _, answeredAsk := range answeredAsks {
		if len(answeredAsk.Abstract) > 0 {
			continue
		}
		amount++
	}

	return amount
}
