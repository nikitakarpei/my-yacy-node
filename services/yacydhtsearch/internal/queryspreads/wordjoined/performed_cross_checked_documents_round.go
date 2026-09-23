package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedCrossCheckedDocumentsRound struct {
	AmountOfDocumentsSentForCrossChecking              int
	AmountOfCrossCheckCandidatesNoPeerTook             int
	AmountOfCrossCheckCandidatesRuledOutByAFullListing int
	AmountOfEmptyCrossCheckedDocumentsAnswers          int
	AmountOfJoinedDocuments                            int
	AmountOfJoinedDocumentsFoundOnlyByCrossChecking    int
	JudgedPeers                                        []peerjudgements.JudgedPeer
}

func performedCrossCheckedDocumentsRoundFrom(
	round crossCheckedDocumentsRound,
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	judgedPeers []peerjudgements.JudgedPeer,
	joinedDocuments distinctDocuments,
) PerformedCrossCheckedDocumentsRound {
	return PerformedCrossCheckedDocumentsRound{
		AmountOfDocumentsSentForCrossChecking: amountOfDocumentsSentForCrossCheckingAcross(
			round.asks,
		),
		AmountOfCrossCheckCandidatesNoPeerTook: amountOfCrossCheckCandidatesAcross(
			round.candidates.ofPartlyListedWordPartitions,
		) -
			amountOfDocumentsSentForCrossCheckingAcross(
				round.asks,
			),
		AmountOfCrossCheckCandidatesRuledOutByAFullListing: amountOfCrossCheckCandidatesAcross(
			round.candidates.ofFullyListedWordPartitions,
		),
		AmountOfEmptyCrossCheckedDocumentsAnswers: amountOfEmptyCrossCheckedDocumentsAnswers(
			round.answeredAsks,
		),
		AmountOfJoinedDocuments: len(joinedDocuments),
		AmountOfJoinedDocumentsFoundOnlyByCrossChecking: amountOfJoinedDocumentsFoundOnlyByCrossChecking(
			joinedDocuments,
			matchedAndHeldDocumentsRound,
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
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
) int {
	documentsOfEveryQueryWordListedByPeers := matchedAndHeldDocumentsRound.
		documentsListedByPeersPerQueryWord().documentsOfEveryQueryWord()

	return len(joinedDocuments) - len(documentsOfEveryQueryWordListedByPeers)
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

func amountOfEmptyCrossCheckedDocumentsAnswers(
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
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
